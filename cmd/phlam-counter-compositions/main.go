package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/LynnColeArt/phlam/runtime/learned"
	"github.com/LynnColeArt/phlam/runtime/reference"
	"github.com/LynnColeArt/phlam/semantics"
)

type observation struct {
	Sequence     learned.Sequence `json:"sequence"`
	Supported    bool             `json:"within_certified_domain"`
	FirstOutside int              `json:"first_outside_step"`
	Results      []int64          `json:"oracle_results"`
	Writes       []int64          `json:"oracle_ordered_writes"`
	Final        int64            `json:"oracle_final"`
	Version      uint64           `json:"oracle_version"`
	TraceHash    string           `json:"oracle_observed_trace_sha256"`
}
type metric struct {
	N    int     `json:"samples"`
	Mean float64 `json:"mean_ns"`
	P50  int64   `json:"p50_ns"`
	P95  int64   `json:"p95_ns"`
}

func measure(v []int64) metric {
	if len(v) == 0 {
		return metric{}
	}
	values := append([]int64(nil), v...)
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	sum := 0.0
	for _, n := range values {
		sum += float64(n)
	}
	return metric{len(v), sum / float64(len(v)), values[(len(v)-1)/2], values[int(math.Ceil(float64(len(v))*.95))-1]}
}

type response struct {
	Schema         string                       `json:"schema"`
	Engine         string                       `json:"engine"`
	Boundary       string                       `json:"boundary"`
	LoadNS         int64                        `json:"load_ns"`
	Certificate    learned.Certificate          `json:"certificate"`
	PreflightError string                       `json:"preflight_error"`
	Outcomes       []learned.CompositionOutcome `json:"outcomes"`
}
type counts struct {
	Scheduled         int `json:"scheduled"`
	Evaluated         int `json:"evaluated"`
	Supported         int `json:"supported"`
	Accepted          int `json:"accepted"`
	WrongAccepted     int `json:"wrong_accepted"`
	SupportedRejected int `json:"supported_rejected"`
	OutsideRejected   int `json:"outside_domain_rejected"`
}
type stats struct {
	Counts          counts                       `json:"accuracy"`
	Lengths         map[int]*counts              `json:"lengths"`
	Timing          map[string]metric            `json:"timings_all"`
	SupportedTiming map[string]metric            `json:"timings_supported"`
	Work            learned.CompositionWork      `json:"runtime_checker_work_total"`
	CheckerCalls    int                          `json:"runtime_concrete_checker_calls"`
	LoadNS          int64                        `json:"load_ns"`
	Certificate     learned.Certificate          `json:"certificate"`
	PreflightError  string                       `json:"preflight_error,omitempty"`
	Examples        []learned.CompositionOutcome `json:"first_rejections,omitempty"`
}
type report struct {
	Schema                   string           `json:"schema"`
	SourceHash               string           `json:"source_sha256"`
	ArtifactHash             string           `json:"reused_frozen_artifact_sha256"`
	OriginalTrainingHash     string           `json:"original_single_step_training_sha256"`
	ObservationsHash         string           `json:"observations_sha256"`
	ImplementationHash       string           `json:"implementation_sha256"`
	WorkerHash               string           `json:"worker_binary_sha256"`
	ABI                      string           `json:"composition_abi"`
	Toolchain                string           `json:"toolchain"`
	Platform                 string           `json:"platform"`
	NewTrainingExamples      int              `json:"new_training_examples"`
	OriginalTrainingExamples int              `json:"original_single_step_training_examples"`
	Sequences                int              `json:"unique_sequences"`
	Shapes                   int              `json:"unique_operation_shapes"`
	Supported                int              `json:"supported_sequences"`
	Outside                  int              `json:"outside_domain_sequences"`
	ObservationNS            int64            `json:"oracle_observation_ns"`
	Oracle                   metric           `json:"oracle_composition_including_internal_checks"`
	Repeats                  int              `json:"timing_repeats"`
	SourceFreeFiles          []string         `json:"source_free_directory_files"`
	WorkerGOMAXPROCS         int              `json:"worker_gomaxprocs"`
	Engines                  map[string]stats `json:"engines"`
	CheckerBreakEven         map[string]int64 `json:"estimated_checker_only_break_even_sequences"`
	TotalBreakEven           map[string]int64 `json:"estimated_total_break_even_sequences"`
	Limitations              []string         `json:"limitations"`
}

func sequences() []learned.Sequence {
	out := []learned.Sequence{}
	seen := map[string]bool{}
	appendPattern := func(length int, shape string) {
		for _, initial := range []int64{-48, 0, 48} {
			for _, delta := range []int64{-9, 0, 11} {
				steps := make([]learned.Operation, length)
				for i := range steps {
					steps[i].ID = "Counter.read"
					if shape[i] == 'I' {
						steps[i].ID = "Counter.increment"
						steps[i].Delta = delta
						if i%3 == 1 {
							steps[i].Delta = -delta
						}
					}
				}
				raw, _ := json.Marshal(struct {
					Initial int64
					Steps   []learned.Operation
				}{initial, steps})
				key := learned.Hash(raw)
				if seen[key] {
					continue
				}
				seen[key] = true
				out = append(out, learned.Sequence{ID: key, Split: fmt.Sprintf("untrained_length_%d", length), Initial: initial, Steps: steps})
			}
		}
	}
	for length := 2; length <= 6; length++ {
		for mask := 0; mask < 1<<length; mask++ {
			var shape strings.Builder
			for i := 0; i < length; i++ {
				if mask&(1<<i) != 0 {
					shape.WriteByte('I')
				} else {
					shape.WriteByte('R')
				}
			}
			appendPattern(length, shape.String())
		}
	}
	for _, length := range []int{8, 12, 16} {
		for _, pattern := range []string{"I", "R", "IR", "IIR"} {
			shape := strings.Repeat(pattern, length)
			appendPattern(length, shape[:length])
		}
	}
	return out
}
func observe(source string, s learned.Sequence) (observation, int64, error) {
	o := observation{Sequence: s, Supported: true, FirstOutside: -1, Results: []int64{}, Writes: []int64{}}
	r, err := reference.New(source, reference.Options{CaptureIntegers: true})
	if err != nil {
		return o, 0, err
	}
	h, err := r.Construct("Counter", semantics.Integer(s.Initial))
	if err != nil {
		return o, 0, err
	}
	offset := len(r.Events())
	calls := make([]reference.Invocation, len(s.Steps))
	for i, step := range s.Steps {
		calls[i].Method = "read"
		if step.ID == "Counter.increment" {
			calls[i].Method = "increment"
			calls[i].Args = []semantics.Value{semantics.Integer(step.Delta)}
		}
	}
	start := time.Now()
	values, err := r.Compose(h, calls)
	elapsed := time.Since(start).Nanoseconds()
	if err != nil {
		return o, 0, err
	}
	expected := big.NewInt(s.Initial)
	wrote := false
	for i, value := range values {
		if (expected.Int64() < -64 || expected.Int64() > 64) && o.Supported {
			o.Supported = false
			o.FirstOutside = i
		}
		if s.Steps[i].ID == "Counter.increment" {
			expected.Add(expected, big.NewInt(s.Steps[i].Delta))
			wrote = true
		}
		if !expected.IsInt64() || value != semantics.Integer(expected.Int64()) {
			return o, 0, fmt.Errorf("oracle differs from independent composition sum")
		}
		o.Results = append(o.Results, value.Int)
	}
	objects := r.Objects()
	o.Final = objects[0].Fields["value"].Int
	o.Version = objects[0].Version
	version := uint64(1)
	if wrote {
		version = 2
	}
	if o.Final != expected.Int64() || o.Version != version {
		return o, 0, fmt.Errorf("oracle composition state/version mismatch")
	}
	events := r.Events()[offset:]
	checks := 0
	for _, e := range events {
		if e.Operation == "stage_write" {
			o.Writes = append(o.Writes, e.Integers["value"])
		}
		if e.Operation == "contract_validated" {
			checks++
		}
	}
	if checks != len(s.Steps) {
		return o, 0, fmt.Errorf("oracle missing per-method checks")
	}
	raw, err := json.Marshal(events)
	if err != nil {
		return o, 0, err
	}
	o.TraceHash = learned.Hash(raw)
	return o, elapsed, nil
}
func addRelation(a *learned.CompositionWork, b learned.CompositionWork) {
	a.DomainChecks += b.DomainChecks
	a.PredictionChecks += b.PredictionChecks
	a.CertificateUses += b.CertificateUses
	a.Relation.TermEvaluations += b.Relation.TermEvaluations
	a.Relation.IntegerPrimitives += b.Relation.IntegerPrimitives
	a.Relation.ArgumentChecks += b.Relation.ArgumentChecks
	a.Relation.FieldChecks += b.Relation.FieldChecks
	a.Relation.OrderedWriteChecks += b.Relation.OrderedWriteChecks
}
func summarize(w response, observations []observation) (stats, error) {
	s := stats{Counts: counts{Scheduled: len(observations)}, Lengths: map[int]*counts{}, Timing: map[string]metric{}, SupportedTiming: map[string]metric{}, LoadNS: w.LoadNS, Certificate: w.Certificate, PreflightError: w.PreflightError}
	if w.PreflightError != "" {
		if len(w.Outcomes) != 0 {
			return s, fmt.Errorf("failed preflight ran outcomes")
		}
		return s, nil
	}
	labels := map[string]observation{}
	for _, o := range observations {
		labels[o.Sequence.ID] = o
	}
	all, supported := map[string][]int64{}, map[string][]int64{}
	for i, outcome := range w.Outcomes {
		o, ok := labels[outcome.ID]
		if !ok {
			return s, fmt.Errorf("unknown worker case")
		}
		addRelation(&s.Work, outcome.Work)
		s.CheckerCalls += outcome.Timing.CheckerCalls
		phases := map[string]int64{"proposal": outcome.Timing.ProposalNS, "candidate": outcome.Timing.CandidateNS, "checker": outcome.Timing.CheckerNS, "commit": outcome.Timing.CommitNS, "overhead": outcome.Timing.OverheadNS, "total": outcome.Timing.TotalNS}
		for phase, n := range phases {
			all[phase] = append(all[phase], n)
			if o.Supported {
				supported[phase] = append(supported[phase], n)
			}
		}
		correct := outcome.State == o.Final && outcome.Version == o.Version && slices.Equal(outcome.Results, o.Results) && slices.Equal(outcome.Writes, o.Writes)
		if outcome.Accepted && !correct {
			return s, fmt.Errorf("incorrect accepted composition: %s %s/%s got=%+v want=%+v", outcome.ID, w.Engine, w.Boundary, outcome, o)
		}
		if !outcome.Accepted && (outcome.State != o.Sequence.Initial || outcome.Version != 1 || len(outcome.Results) != 0 || len(outcome.Writes) != 0) {
			return s, fmt.Errorf("rejected composition leaked staged state")
		}
		if i >= len(observations) {
			continue
		}
		length := len(o.Sequence.Steps)
		if s.Lengths[length] == nil {
			s.Lengths[length] = &counts{}
		}
		s.Lengths[length].Scheduled++
		for _, c := range []*counts{&s.Counts, s.Lengths[length]} {
			c.Evaluated++
			if o.Supported {
				c.Supported++
			}
			if outcome.Accepted {
				c.Accepted++
				if !correct {
					c.WrongAccepted++
				}
			} else if o.Supported {
				c.SupportedRejected++
			} else {
				c.OutsideRejected++
			}
		}
		if !outcome.Accepted && len(s.Examples) < 3 {
			s.Examples = append(s.Examples, outcome)
		}
	}
	for phase, v := range all {
		s.Timing[phase] = measure(v)
	}
	for phase, v := range supported {
		s.SupportedTiming[phase] = measure(v)
	}
	return s, nil
}
func jsonBytes(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	return append(b, '\n'), err
}
func observationBytes(observations []observation) ([]byte, error) {
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	for _, o := range observations {
		if err := encoder.Encode(o); err != nil {
			return nil, err
		}
	}
	return b.Bytes(), nil
}
func copyFile(from, to string, mode os.FileMode) error {
	b, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, b, mode)
}
func implementationHash() (string, error) {
	files := []string{"go.mod"}
	for _, root := range []string{"cmd", "compiler", "runtime", "semantics"} {
		if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, ".go") {
				files = append(files, filepath.ToSlash(path))
			}
			return nil
		}); err != nil {
			return "", err
		}
	}
	sort.Strings(files)
	var b bytes.Buffer
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "%s %s\n", path, learned.Hash(data))
	}
	return learned.Hash(b.Bytes()), nil
}
func runWorker(dir, engine, boundary string, sequences []learned.Sequence, repeats int) (response, error) {
	input, err := json.Marshal(struct {
		Sequences []learned.Sequence `json:"sequences"`
		Repeats   int                `json:"repeats"`
	}{sequences, repeats})
	if err != nil {
		return response{}, err
	}
	command := exec.Command(filepath.Join(dir, "worker"), "artifact.json", "trusted-key.hex", engine, boundary)
	command.Dir = dir
	command.Env = []string{"GOMAXPROCS=1"}
	command.Stdin = bytes.NewReader(input)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	raw, err := command.Output()
	if err != nil {
		return response{}, fmt.Errorf("source-free worker: %w %s", err, &stderr)
	}
	var result response
	if err := json.Unmarshal(raw, &result); err != nil {
		return result, err
	}
	if result.PreflightError == "" && len(result.Outcomes) != len(sequences)*repeats {
		return result, fmt.Errorf("worker result count mismatch")
	}
	return result, nil
}
func run() error {
	sourcePath := flag.String("source", "examples/counter/counter.phlam", "oracle source")
	artifactPath := flag.String("artifact", "experiments/counter/run-0001/artifact.json", "existing frozen single-step artifact")
	trainingPath := flag.String("training", "experiments/counter/run-0001/training.json", "original single-step training observations for lineage audit")
	keyPath := flag.String("key", "experiments/counter/run-0001/trusted-key.hex", "research public key")
	workerPath := flag.String("worker", "bin/phlam-counter-compose-worker", "source-free worker executable")
	output := flag.String("out", "experiments/counter-compositions/run-0001", "new output directory")
	repeats := flag.Int("repeats", 10, "timing repetitions")
	flag.Parse()
	if *repeats < 1 || *repeats > 100 {
		return fmt.Errorf("repeats must be 1..100")
	}
	if _, err := os.Stat(*output); !os.IsNotExist(err) {
		return fmt.Errorf("output must not exist")
	}
	key, err := learned.PublicKey(*keyPath)
	if err != nil {
		return err
	}
	if _, err := learned.Load(*artifactPath, key); err != nil {
		return err
	}
	source, err := os.ReadFile(*sourcePath)
	if err != nil {
		return err
	}
	oracle, err := reference.New(string(source), reference.Options{})
	if err != nil {
		return err
	}
	rawArtifact, err := os.ReadFile(*artifactPath)
	if err != nil {
		return err
	}
	var envelope learned.Envelope
	if err := json.Unmarshal(rawArtifact, &envelope); err != nil {
		return err
	}
	var artifact learned.Artifact
	if err := json.Unmarshal(envelope.Payload, &artifact); err != nil {
		return err
	}
	if artifact.SourceHash != oracle.SourceHash() {
		return fmt.Errorf("source and frozen artifact lineage differ")
	}
	trainingBytes, err := os.ReadFile(*trainingPath)
	if err != nil {
		return err
	}
	if learned.Hash(trainingBytes) != artifact.TrainingHash {
		return fmt.Errorf("original training lineage mismatch")
	}
	var originalTraining []learned.Label
	if err := json.Unmarshal(trainingBytes, &originalTraining); err != nil {
		return err
	}
	implementation, err := implementationHash()
	if err != nil {
		return err
	}
	r := report{Schema: "phlam.counter.compositions.experiment.v0", SourceHash: oracle.SourceHash(), ArtifactHash: learned.Hash(rawArtifact), OriginalTrainingHash: artifact.TrainingHash, ImplementationHash: implementation, ABI: learned.CompositionABI, Toolchain: runtime.Version(), Platform: runtime.GOOS + "/" + runtime.GOARCH, OriginalTrainingExamples: len(originalTraining), Repeats: *repeats, WorkerGOMAXPROCS: 1, Engines: map[string]stats{}, CheckerBreakEven: map[string]int64{}, TotalBreakEven: map[string]int64{}, Limitations: []string{
		"No new weights or composition examples were trained; only single-step learned transitions are rolled out.",
		"Schedules are supplied by the caller, not invented or chosen by a learned planner.",
		"Finite-domain certification proves primitive predictions on this worker; composition safety follows by induction with domain guards.",
		"Certification enumerates 4386 primitive inputs once; it is exact work shifted to startup, not free or symbolic verification.",
		"Sequences leaving the certified input domain reject atomically although the oracle can execute them.",
		"Only one Counter instance/slot, read and increment, and 1..16 supplied operations; no branches, aliases, eval or external effects.",
		"Warm timing contains instrumentation; cold load and certification are separate; break-even is an estimate on this workload.",
		"The existing artifact uses a public research signing key, not a production trust identity.",
	}}
	cases := sequences()
	shapes := map[string]bool{}
	observations := make([]observation, 0, len(cases))
	oracleTimes := []int64{}
	start := time.Now()
	for _, s := range cases {
		o, elapsed, err := observe(string(source), s)
		if err != nil {
			return err
		}
		observations = append(observations, o)
		oracleTimes = append(oracleTimes, elapsed)
		if o.Supported {
			r.Supported++
		} else {
			r.Outside++
		}
		var shape strings.Builder
		for _, step := range s.Steps {
			if step.ID == "Counter.increment" {
				shape.WriteByte('I')
			} else {
				shape.WriteByte('R')
			}
		}
		shapes[shape.String()] = true
	}
	r.ObservationNS = time.Since(start).Nanoseconds()
	r.Oracle = measure(oracleTimes)
	r.Sequences = len(cases)
	r.Shapes = len(shapes)
	observationsBytes, err := observationBytes(observations)
	if err != nil {
		return err
	}
	r.ObservationsHash = learned.Hash(observationsBytes)
	dir, err := os.MkdirTemp("", "phlam-compositions-source-free-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	for _, file := range []struct {
		From, To string
		Mode     os.FileMode
	}{{*workerPath, "worker", 0755}, {*artifactPath, "artifact.json", 0444}, {*keyPath, "trusted-key.hex", 0644}} {
		if err := copyFile(file.From, filepath.Join(dir, file.To), file.Mode); err != nil {
			return err
		}
	}
	binary, err := os.ReadFile(filepath.Join(dir, "worker"))
	if err != nil {
		return err
	}
	r.WorkerHash = learned.Hash(binary)
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		r.SourceFreeFiles = append(r.SourceFreeFiles, f.Name())
	}
	for _, engine := range []string{"learned", "lookup", "zero", "shuffled"} {
		for _, boundary := range []string{"concrete", "certified"} {
			n := *repeats
			if engine == "zero" || engine == "shuffled" {
				n = 1
			}
			w, err := runWorker(dir, engine, boundary, cases, n)
			if err != nil {
				return err
			}
			s, err := summarize(w, observations)
			if err != nil {
				return err
			}
			r.Engines[engine+"/"+boundary] = s
		}
	}
	for _, engine := range []string{"learned", "lookup"} {
		concrete := r.Engines[engine+"/concrete"]
		certified := r.Engines[engine+"/certified"]
		for _, kind := range []string{"checker", "total"} {
			saving := concrete.Timing[kind].Mean - certified.Timing[kind].Mean
			if saving > 0 {
				estimate := int64(math.Ceil(float64(certified.Certificate.BuildNS) / saving))
				if kind == "checker" {
					r.CheckerBreakEven[engine] = estimate
				} else {
					r.TotalBreakEven[engine] = estimate
				}
			}
		}
	}
	if err := os.MkdirAll(*output, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*output, "observations.jsonl"), observationsBytes, 0644); err != nil {
		return err
	}
	raw, err := jsonBytes(r)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*output, "report.json"), raw, 0644); err != nil {
		return err
	}
	fmt.Printf("saved %s: sequences=%d shapes=%d supported=%d outside=%d learned concrete=%d certified=%d\n", *output, r.Sequences, r.Shapes, r.Supported, r.Outside, r.Engines["learned/concrete"].Counts.Accepted, r.Engines["learned/certified"].Counts.Accepted)
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
