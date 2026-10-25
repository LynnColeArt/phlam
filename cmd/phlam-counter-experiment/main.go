// Train/evaluate orchestration owns source and the oracle. The separately built
// worker owns neither. No oracle result is passed to a worker as a proposal.
package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/LynnColeArt/phlam/compiler/contracts"
	"github.com/LynnColeArt/phlam/runtime/learned"
	"github.com/LynnColeArt/phlam/runtime/reference"
	"github.com/LynnColeArt/phlam/semantics"
)

const seed int64 = 20261025

type metric struct {
	Samples int     `json:"samples"`
	Mean    float64 `json:"mean_ns"`
	P50     int64   `json:"p50_ns"`
	P95     int64   `json:"p95_ns"`
}

func measure(values []int64) metric {
	if len(values) == 0 {
		return metric{}
	}
	v := append([]int64(nil), values...)
	sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	sum := 0.0
	for _, n := range v {
		sum += float64(n)
	}
	return metric{len(v), sum / float64(len(v)), v[(len(v)-1)/2], v[int(math.Ceil(float64(len(v))*.95))-1]}
}

type counts struct {
	Cases         int `json:"cases"`
	RawCorrect    int `json:"raw_correct"`
	Accepted      int `json:"accepted"`
	AcceptedWrong int `json:"accepted_wrong"`
	Rejected      int `json:"rejected"`
}
type workerResponse struct {
	Schema         string            `json:"schema"`
	Engine         string            `json:"engine"`
	LoadNS         int64             `json:"load_ns"`
	AllocatedBytes uint64            `json:"allocated_bytes"`
	Outcomes       []learned.Outcome `json:"outcomes"`
}
type stats struct {
	CheckerWork    contracts.Work     `json:"checker_work_total"`
	Counts         counts             `json:"accuracy"`
	Splits         map[string]*counts `json:"splits"`
	Timings        map[string]metric  `json:"timings"`
	CheckerCalls   int                `json:"checker_calls"`
	LoadNS         int64              `json:"load_ns"`
	AllocatedBytes uint64             `json:"load_allocated_bytes"`
	Examples       []learned.Outcome  `json:"first_rejections,omitempty"`
}
type report struct {
	Schema             string           `json:"schema"`
	Seed               int64            `json:"seed"`
	SourceHash         string           `json:"source_sha256"`
	ArtifactHash       string           `json:"artifact_sha256"`
	TrainingHash       string           `json:"training_sha256"`
	SplitsHash         string           `json:"splits_sha256"`
	ImplementationHash string           `json:"implementation_sha256"`
	Toolchain          string           `json:"toolchain"`
	Platform           string           `json:"platform"`
	WorkerGOMAXPROCS   int              `json:"worker_gomaxprocs"`
	TrainingCases      int              `json:"training_cases"`
	TestCases          int              `json:"test_cases"`
	Parameters         int              `json:"learned_parameters"`
	InitialLoss        float64          `json:"initial_normalized_mse"`
	FinalLoss          float64          `json:"final_normalized_mse"`
	DataGenerationNS   int64            `json:"oracle_data_generation_ns"`
	TrainingNS         int64            `json:"training_ns"`
	ModelReproducible  bool             `json:"bit_identical_retraining"`
	SourceFree         bool             `json:"source_free_worker"`
	WorkerFiles        []string         `json:"source_free_directory_files"`
	WorkerHash         string           `json:"worker_binary_sha256"`
	ArtifactBytes      int              `json:"artifact_bytes"`
	Repeats            int              `json:"timing_repeats"`
	OracleInvoke       metric           `json:"oracle_invoke_including_internal_checker"`
	Engines            map[string]stats `json:"engines"`
	Limitations        []string         `json:"limitations"`
}

func cases() []learned.Case {
	out := []learned.Case{}
	for s := int64(-64); s <= 64; s++ {
		for _, op := range []string{"Counter.read", "Counter.increment"} {
			lo, hi := int64(-16), int64(16)
			if op == "Counter.read" {
				lo = 0
				hi = 0
			}
			for d := lo; d <= hi; d++ {
				split := "interpolation"
				if s < -32 || s > 32 || d < -8 || d > 8 {
					split = "extrapolation"
				}
				if s >= -32 && s <= 32 && s%4 == 0 && (op == "Counter.read" || d >= -8 && d <= 8 && d%2 == 0) {
					split = "train"
				}
				out = append(out, learned.Case{ID: fmt.Sprintf("%s:%d:%d", op, s, d), Split: split, Operation: op, State: s, Delta: d})
			}
		}
	}
	return out
}
func observe(source string, c learned.Case) (learned.Label, int64, error) {
	r, err := reference.New(source, reference.Options{CaptureIntegers: c.Split == "train"})
	if err != nil {
		return learned.Label{}, 0, err
	}
	handle, err := r.Construct("Counter", semantics.Integer(c.State))
	if err != nil {
		return learned.Label{}, 0, err
	}
	method := "read"
	args := []semantics.Value{}
	write := false
	if c.Operation == "Counter.increment" {
		method = "increment"
		args = append(args, semantics.Integer(c.Delta))
		write = true
	}
	start := time.Now()
	value, err := r.Call(handle, method, args...)
	elapsed := time.Since(start).Nanoseconds()
	if err != nil {
		return learned.Label{}, 0, err
	}
	objects := r.Objects()
	next := objects[0].Fields["value"].Int
	expected := big.NewInt(c.State)
	if write {
		expected.Add(expected, big.NewInt(c.Delta))
	}
	if !expected.IsInt64() || value != semantics.Integer(expected.Int64()) || next != expected.Int64() || objects[0].Version != map[bool]uint64{true: 2, false: 1}[write] {
		return learned.Label{}, 0, fmt.Errorf("oracle disagreed with independent exact sum for %s", c.ID)
	}
	events := r.Events()
	checks := 0
	for _, e := range events {
		if e.Operation == "contract_validated" {
			checks++
		}
	}
	if checks != 2 {
		return learned.Label{}, 0, fmt.Errorf("oracle lacks expected observed contract checks")
	}
	raw, err := json.Marshal(events)
	if err != nil {
		return learned.Label{}, 0, err
	}
	return learned.Label{Case: c, Next: next, Result: value.Int, Write: write, TraceHash: learned.Hash(raw)}, elapsed, nil
}
func jsonBytes(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	return append(b, '\n'), err
}
func save(path string, v any) error {
	b, err := jsonBytes(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
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
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, ".go") {
				files = append(files, filepath.ToSlash(path))
			}
			return nil
		})
		if err != nil {
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
func worker(dir, engine string, test []learned.Case, repeats int) (workerResponse, error) {
	input, err := json.Marshal(struct {
		Cases   []learned.Case `json:"cases"`
		Repeats int            `json:"repeats"`
	}{test, repeats})
	if err != nil {
		return workerResponse{}, err
	}
	command := exec.Command(filepath.Join(dir, "worker"), "artifact.json", "trusted-key.hex", engine)
	command.Dir = dir
	command.Env = []string{"GOMAXPROCS=1"}
	command.Stdin = bytes.NewReader(input)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return workerResponse{}, fmt.Errorf("source-free worker: %w: %s", err, &stderr)
	}
	var response workerResponse
	err = json.Unmarshal(output, &response)
	if err == nil && len(response.Outcomes) != len(test)*repeats {
		return response, fmt.Errorf("worker result count mismatch")
	}
	return response, err
}
func summarize(w workerResponse, labels map[string]learned.Label, testCases int) stats {
	s := stats{Splits: map[string]*counts{}, Timings: map[string]metric{}, LoadNS: w.LoadNS, AllocatedBytes: w.AllocatedBytes}
	values := map[string][]int64{}
	for i, o := range w.Outcomes {
		s.CheckerCalls += o.Timing.CheckerCalls
		s.CheckerWork.TermEvaluations += o.CheckerWork.TermEvaluations
		s.CheckerWork.IntegerPrimitives += o.CheckerWork.IntegerPrimitives
		s.CheckerWork.ArgumentChecks += o.CheckerWork.ArgumentChecks
		s.CheckerWork.FieldChecks += o.CheckerWork.FieldChecks
		s.CheckerWork.OrderedWriteChecks += o.CheckerWork.OrderedWriteChecks
		for key, n := range map[string]int64{"proposal": o.Timing.ProposalNS, "candidate": o.Timing.CandidateNS, "checker": o.Timing.CheckerNS, "commit": o.Timing.CommitNS, "overhead": o.Timing.OverheadNS, "total": o.Timing.TotalNS} {
			values[key] = append(values[key], n)
		}
		if i >= testCases {
			continue
		}
		l := labels[o.ID]
		if s.Splits[l.Case.Split] == nil {
			s.Splits[l.Case.Split] = &counts{}
		}
		for _, c := range []*counts{&s.Counts, s.Splits[l.Case.Split]} {
			c.Cases++
			raw := int64(math.Round(o.Scores[0]*learned.Scale)) == l.Next && int64(math.Round(o.Scores[1]*learned.Scale)) == l.Result && (o.Scores[2] >= .5) == l.Write
			if raw {
				c.RawCorrect++
			}
			if o.Accepted {
				c.Accepted++
				if o.State != l.Next || o.Result != l.Result || o.Write != l.Write {
					c.AcceptedWrong++
				}
			} else {
				c.Rejected++
			}
		}
		if !o.Accepted && len(s.Examples) < 3 {
			s.Examples = append(s.Examples, o)
		}
	}
	for key, v := range values {
		s.Timings[key] = measure(v)
	}
	return s
}
func run() error {
	sourcePath := flag.String("source", "examples/counter/counter.phlam", "oracle training source")
	workerPath := flag.String("worker", "bin/phlam-counter-worker", "separately built source-free worker")
	output := flag.String("out", "experiments/counter/run-0001", "new result directory (must not exist)")
	repeats := flag.Int("repeats", 10, "timing repetitions per held-out case")
	flag.Parse()
	if *repeats < 1 || *repeats > 100 {
		return fmt.Errorf("repeats must be 1..100")
	}
	if _, err := os.Stat(*output); !os.IsNotExist(err) {
		return fmt.Errorf("output must be a new directory")
	}
	source, err := os.ReadFile(*sourcePath)
	if err != nil {
		return err
	}
	oracle, err := reference.New(string(source), reference.Options{})
	if err != nil {
		return err
	}
	implementation, err := implementationHash()
	if err != nil {
		return err
	}
	r := report{Schema: "phlam.counter.experiment.v0", Seed: seed, SourceHash: oracle.SourceHash(), ImplementationHash: implementation, Toolchain: runtime.Version(), Platform: runtime.GOOS + "/" + runtime.GOARCH, WorkerGOMAXPROCS: 1, Parameters: 12, Repeats: *repeats, Engines: map[string]stats{}, Limitations: []string{
		"One Counter program; no held-out program templates or compositions.",
		"Four engineered features include operation ID; reads encode absent delta as zero.",
		"Numeric predictions are rounded; safety/correctness depends on concrete exact checking.",
		"The checker repeats the operational arithmetic; there is no symbolic term proof or efficiency claim.",
		"Bounded exhaustive domain excludes int64 extremes, eval, aliases, tenants and external effects.",
		"Structural lookup + exact primitives already solves this task without learning.",
		"Signing uses a publicly reproducible research key, not a production trust root.",
		"Checker instrumentation and in-process stage timers add overhead; load allocation is TotalAlloc delta, not RSS/peak memory.",
		"Oracle invoke includes its internal checker, trace and heap overhead; parse/construction excluded.",
	}}
	all := cases()
	train := []learned.Label{}
	test := []learned.Case{}
	labels := map[string]learned.Label{}
	oracleTimes := []int64{}
	start := time.Now()
	for _, c := range all {
		l, elapsed, err := observe(string(source), c)
		if err != nil {
			return err
		}
		labels[c.ID] = l
		if c.Split == "train" {
			train = append(train, l)
		} else {
			test = append(test, c)
			oracleTimes = append(oracleTimes, elapsed)
		}
	}
	r.DataGenerationNS = time.Since(start).Nanoseconds()
	r.TrainingCases = len(train)
	r.TestCases = len(test)
	r.OracleInvoke = measure(oracleTimes)
	trainBytes, err := jsonBytes(train)
	if err != nil {
		return err
	}
	splitBytes, err := jsonBytes(all)
	if err != nil {
		return err
	}
	r.TrainingHash = learned.Hash(trainBytes)
	r.SplitsHash = learned.Hash(splitBytes)
	start = time.Now()
	model, initial, final, err := learned.Train(train, seed)
	r.TrainingNS = time.Since(start).Nanoseconds()
	if err != nil {
		return err
	}
	r.InitialLoss = initial
	r.FinalLoss = final
	again, _, _, err := learned.Train(train, seed)
	if err != nil {
		return err
	}
	r.ModelReproducible = model == again
	bundle := contracts.Bundle{Schema: "phlam.contracts.v0", SourceHash: r.SourceHash, Methods: []contracts.Contract{}}
	for _, c := range oracle.Contracts().Methods {
		if c.ID == "Counter.increment" || c.ID == "Counter.read" {
			bundle.Methods = append(bundle.Methods, c)
		}
	}
	artifact := learned.Artifact{Schema: "phlam.counter.learned.v0", Architecture: "linear-4x3-round-v0", PrimitiveABI: "phlam.int64.v0", Domain: "state[-64,64];delta[-16,16]", Seed: seed, Epochs: learned.Epochs, LearningRate: learned.Rate, SourceHash: r.SourceHash, TrainingHash: r.TrainingHash, SplitsHash: r.SplitsHash, ImplementationHash: implementation, Toolchain: r.Toolchain, SigningPolicy: "public_reproducible_research_key_only", Model: model, Contracts: bundle}
	// Publicly reproducible key for this research fixture; not a secret credential.
	signingSeed := sha256.Sum256([]byte("PHLAM Counter experiment public research signing key v0"))
	key := ed25519.NewKeyFromSeed(signingSeed[:])
	dir, err := os.MkdirTemp("", "phlam-counter-source-free-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	artifactPath := filepath.Join(dir, "artifact.json")
	if err := learned.Freeze(artifactPath, artifact, key); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "trusted-key.hex"), []byte(hex.EncodeToString(key.Public().(ed25519.PublicKey))), 0644); err != nil {
		return err
	}
	if err := copyFile(*workerPath, filepath.Join(dir, "worker"), 0755); err != nil {
		return err
	}
	artifactBytes, err := os.ReadFile(artifactPath)
	if err != nil {
		return err
	}
	r.ArtifactHash = learned.Hash(artifactBytes)
	r.ArtifactBytes = len(artifactBytes)
	binary, err := os.ReadFile(filepath.Join(dir, "worker"))
	if err != nil {
		return err
	}
	r.WorkerHash = learned.Hash(binary)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		r.WorkerFiles = append(r.WorkerFiles, e.Name())
	}
	r.SourceFree = len(r.WorkerFiles) == 3
	for _, engine := range []string{"learned", "lookup", "zero", "shuffled"} {
		n := *repeats
		if engine == "zero" || engine == "shuffled" {
			n = 1
		}
		w, err := worker(dir, engine, test, n)
		if err != nil {
			return err
		}
		s := summarize(w, labels, len(test))
		r.Engines[engine] = s
		if s.Counts.AcceptedWrong > 0 {
			return fmt.Errorf("exact checker accepted an incorrect %s transition", engine)
		}
	}
	if err := os.MkdirAll(*output, 0755); err != nil {
		return err
	}
	if err := copyFile(artifactPath, filepath.Join(*output, "artifact.json"), 0444); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(dir, "trusted-key.hex"), filepath.Join(*output, "trusted-key.hex"), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*output, "training.json"), trainBytes, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*output, "splits.json"), splitBytes, 0644); err != nil {
		return err
	}
	if err := save(filepath.Join(*output, "report.json"), r); err != nil {
		return err
	}
	fmt.Printf("saved %s: train=%d test=%d learned accepted=%d lookup accepted=%d zero accepted=%d shuffled accepted=%d\n", *output, len(train), len(test), r.Engines["learned"].Counts.Accepted, r.Engines["lookup"].Counts.Accepted, r.Engines["zero"].Counts.Accepted, r.Engines["shuffled"].Counts.Accepted)
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
