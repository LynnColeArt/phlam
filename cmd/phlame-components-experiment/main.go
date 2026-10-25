package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"time"

	compile "github.com/LynnColeArt/phlam/compiler/components"
	abi "github.com/LynnColeArt/phlam/runtime/components"
	"github.com/LynnColeArt/phlam/runtime/reference"
	"github.com/LynnColeArt/phlam/runtime/trace"
	"github.com/LynnColeArt/phlam/semantics"
)

type config struct {
	Alias, Class, Source   string
	Domain                 abi.Domain
	Seed                   int64
	StateStride, ArgStride int64
	Branching              bool
}

var configs = []config{
	{"counter", "Counter", "examples/counter/counter.phlam", abi.Domain{StateMin: -16, StateMax: 16, ArgMin: -4, ArgMax: 4}, 101, 8, 2, false},
	{"accumulator", "Accumulator", "examples/accumulator/accumulator.phlam", abi.Domain{StateMin: -16, StateMax: 16, ArgMin: -8, ArgMax: 8}, 202, 4, 4, false},
}

func branchingConfig() config {
	return config{"accumulator", "SaturatingAccumulator", "examples/saturating-accumulator/saturating-accumulator.phlam", abi.Domain{StateMin: -16, StateMax: 16, ArgMin: -8, ArgMax: 8}, 404, 2, 4, true}
}

type primitive struct {
	Operation  int            `json:"operation_index"`
	State      int64          `json:"state"`
	Arg        int64          `json:"argument"`
	Accepted   bool           `json:"source_accepted"`
	Prediction abi.Prediction `json:"prediction"`
	Training   bool           `json:"training"`
	TraceHash  string         `json:"observed_trace_sha256"`
}
type rawPrediction struct {
	Operation int            `json:"operation_index"`
	State     int64          `json:"state"`
	Arg       int64          `json:"argument"`
	Training  bool           `json:"training"`
	Target    abi.Prediction `json:"source_target"`
	Scores    [3]float64     `json:"raw_scores"`
	Decoded   abi.Prediction `json:"decoded_prediction"`
	Correct   bool           `json:"correct"`
	Error     string         `json:"error,omitempty"`
}

func rawPredictions(a abi.Artifact, observations []primitive) []rawPrediction {
	out := []rawPrediction{}
	for _, o := range observations {
		if !o.Accepted {
			continue
		}
		scores := a.Model.Predict(len(a.Contract.Operations), o.Operation, o.State, o.Arg)
		p, err := abi.Decode(scores)
		row := rawPrediction{Operation: o.Operation, State: o.State, Arg: o.Arg, Training: o.Training, Target: o.Prediction, Scores: scores, Decoded: p, Correct: err == nil && p == o.Prediction}
		if err != nil {
			row.Error = err.Error()
		}
		out = append(out, row)
	}
	return out
}

type moduleReport struct {
	RawPredictionsHash     string         `json:"raw_predictions_sha256"`
	ModelKind              string         `json:"model_kind"`
	MaxHeldoutIntegerError float64        `json:"max_heldout_integer_score_error"`
	RawTrainingCorrect     int            `json:"raw_training_correct"`
	RawHeldoutCorrect      int            `json:"raw_heldout_correct"`
	HeldoutCases           int            `json:"heldout_cases"`
	BranchCorrect          map[string]int `json:"raw_branch_correct"`
	BranchCases            map[string]int `json:"branch_cases"`
	AffineHeldoutCorrect   int            `json:"affine_baseline_heldout_correct"`
	Parameters             int            `json:"learned_parameters"`
	SourceHash             string         `json:"source_sha256"`
	Root                   string         `json:"module_sha256"`
	TrainingHash           string         `json:"training_sha256"`
	VerificationHash       string         `json:"oracle_domain_sha256"`
	Seed                   int64          `json:"seed"`
	TrainingSamples        int            `json:"training_samples"`
	PrimitiveCases         int            `json:"primitive_cases"`
	RejectedInputs         int            `json:"source_rejected_inputs"`
	InitialLoss            float64        `json:"initial_mse"`
	FinalLoss              float64        `json:"final_mse"`
	TrainingNS             int64          `json:"training_ns"`
	BitIdentical           bool           `json:"bit_identical_retraining"`
	Bytes                  int            `json:"artifact_bytes"`
}
type observation struct {
	Events           []trace.Event           `json:"source_trace_events,omitempty"`
	ObservedBranches map[string]int          `json:"observed_saturation_branches,omitempty"`
	Transaction      abi.Transaction         `json:"transaction"`
	Category         string                  `json:"category"`
	OracleAccepted   bool                    `json:"oracle_accepted"`
	InScope          bool                    `json:"within_certified_coarse_domains"`
	OracleStates     map[string]abi.Instance `json:"oracle_states"`
	OracleResults    []abi.Return            `json:"oracle_results,omitempty"`
	OracleWrites     []abi.Write             `json:"oracle_writes,omitempty"`
	TraceHash        string                  `json:"observed_trace_sha256"`
	OracleError      string                  `json:"oracle_error,omitempty"`
}
type metric struct {
	N    int     `json:"samples"`
	Mean float64 `json:"mean_ns"`
	P50  int64   `json:"p50_ns"`
	P95  int64   `json:"p95_ns"`
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

type response struct {
	Schema         string                     `json:"schema"`
	LoadNS         int64                      `json:"load_ns"`
	Certificates   map[string]abi.Certificate `json:"certificates"`
	PreflightError string                     `json:"preflight_error"`
	Outcomes       []abi.Outcome              `json:"outcomes"`
}
type counts struct {
	Scheduled         int `json:"scheduled"`
	Evaluated         int `json:"evaluated"`
	Accepted          int `json:"accepted"`
	WrongAccepted     int `json:"wrong_accepted"`
	SupportedRejected int `json:"supported_rejected"`
	NativeRejected    int `json:"native_rejected"`
	ScopeRejected     int `json:"scope_rejected"`
}
type engineReport struct {
	Counts         counts                     `json:"accuracy"`
	Certificates   map[string]abi.Certificate `json:"certificates"`
	PreflightError string                     `json:"preflight_error,omitempty"`
	LoadNS         int64                      `json:"load_ns"`
	Timing         map[string]metric          `json:"timings"`
	Work           abi.Measurements           `json:"runtime_checker_work"`
	LookupWork     abi.Work                   `json:"lookup_application_work"`
}
type report struct {
	Schema             string                  `json:"schema"`
	ABI                string                  `json:"abi"`
	ImplementationHash string                  `json:"implementation_sha256"`
	OracleSourceHash   string                  `json:"combined_oracle_source_sha256"`
	Toolchain          string                  `json:"toolchain"`
	Platform           string                  `json:"platform"`
	WorkerHash         string                  `json:"worker_sha256"`
	Modules            map[string]moduleReport `json:"independent_modules"`
	Cases              int                     `json:"transactions"`
	Categories         map[string]int          `json:"oracle_categories"`
	ObservationHash    string                  `json:"observations_sha256"`
	Oracle             metric                  `json:"oracle_transaction_time"`
	Repeats            int                     `json:"timing_repeats"`
	SourceFreeFiles    []string                `json:"source_free_directory_files"`
	Engines            map[string]engineReport `json:"engines"`
	BreakEven          map[string]int64        `json:"estimated_total_break_even_transactions"`
	Limitations        []string                `json:"limitations"`
}

func jsonBytes(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	return append(b, '\n'), err
}
func lines[T any](values []T) ([]byte, error) {
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	for _, v := range values {
		if err := e.Encode(v); err != nil {
			return nil, err
		}
	}
	return b.Bytes(), nil
}
func save(path string, v any) error {
	b, err := jsonBytes(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
func copyFile(from, to string, mode os.FileMode) error {
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, data, mode)
}
func implementationHash() (string, error) {
	files := []string{"go.mod"}
	for _, root := range []string{"cmd", "compiler", "runtime", "semantics"} {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && filepath.Ext(path) == ".go" {
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
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "%s %s\n", file, abi.Hash(data))
	}
	return abi.Hash(b.Bytes()), nil
}
func trainModule(c config, implementation string) (abi.Artifact, []abi.Sample, []primitive, moduleReport, error) {
	var info moduleReport
	source, err := os.ReadFile(c.Source)
	if err != nil {
		return abi.Artifact{}, nil, nil, info, err
	}
	contract, err := compile.Compile(string(source), c.Class)
	if err != nil {
		return abi.Artifact{}, nil, nil, info, err
	}
	info.SourceHash = contract.SourceHash
	info.Seed = c.Seed
	info.Parameters = 3 * (3 + len(contract.Operations))
	samples := []abi.Sample{}
	observed := []primitive{}
	for index, op := range contract.Operations {
		for state := c.Domain.StateMin; state <= c.Domain.StateMax; state++ {
			lo, hi := c.Domain.ArgMin, c.Domain.ArgMax
			if op.Arity == 0 {
				lo, hi = 0, 0
			}
			for arg := lo; arg <= hi; arg++ {
				r, err := reference.New(string(source), reference.Options{CaptureIntegers: true})
				if err != nil {
					return abi.Artifact{}, nil, nil, info, err
				}
				handle, err := r.Construct(c.Class, semantics.Integer(state))
				if err != nil {
					return abi.Artifact{}, nil, nil, info, err
				}
				args := []semantics.Value{}
				if op.Arity == 1 {
					args = append(args, semantics.Integer(arg))
				}
				value, err := r.Call(handle, op.Name, args...)
				if err != nil {
					return abi.Artifact{}, nil, nil, info, err
				}
				object := r.Objects()[0]
				accepted := value.Kind == "int"
				if !accepted && value != semantics.Boolean(false) {
					return abi.Artifact{}, nil, nil, info, fmt.Errorf("unexpected source return")
				}
				prediction := abi.Prediction{Next: object.Fields[contract.Slot].Int, Result: value.Int, Write: object.Version == 2}
				var work abi.Work
				expected, err := op.Evaluate(state, arg, &work)
				if err != nil || expected.Accepted != accepted || accepted && prediction != expected.Prediction || !accepted && (object.Version != 1 || prediction.Next != state) {
					return abi.Artifact{}, nil, nil, info, fmt.Errorf("source/contract disagreement: %s %s state %d arg %d: %v", c.Class, op.Name, state, arg, err)
				}
				// A second independently stated mathematical check prevents shared frontend
				// interpretation errors from being mistaken for the intended demonstration.
				want := state
				nativeOK := true
				if op.Arity == 1 {
					want += arg
					if c.Branching {
						if want < -16 {
							want = -16
						}
						if want > 16 {
							want = 16
						}
					} else if c.Alias == "accumulator" {
						nativeOK = arg >= -8 && arg <= 8 && want >= -16 && want <= 16
					}
				}
				if accepted != nativeOK || accepted && (prediction.Next != want || prediction.Result != want || prediction.Write != (op.Arity == 1)) {
					return abi.Artifact{}, nil, nil, info, fmt.Errorf("source differs from independent arithmetic oracle")
				}
				trace, _ := json.Marshal(r.Events())
				traceHash := abi.Hash(trace)
				training := accepted && state%c.StateStride == 0 && (op.Arity == 0 || arg%c.ArgStride == 0)
				observed = append(observed, primitive{Operation: index, State: state, Arg: arg, Accepted: accepted, Prediction: prediction, Training: training, TraceHash: traceHash})
				if training {
					samples = append(samples, abi.Sample{Operation: index, State: state, Arg: arg, Prediction: prediction, TraceHash: traceHash})
				}
				if !accepted {
					info.RejectedInputs++
				}
			}
		}
	}
	sampleBytes, err := jsonBytes(samples)
	if err != nil {
		return abi.Artifact{}, nil, nil, info, err
	}
	observationBytes, err := lines(observed)
	if err != nil {
		return abi.Artifact{}, nil, nil, info, err
	}
	info.TrainingHash = abi.Hash(sampleBytes)
	info.VerificationHash = abi.Hash(observationBytes)
	info.TrainingSamples = len(samples)
	info.PrimitiveCases = len(observed)
	start := time.Now()
	trainer := abi.Train
	epochs := abi.Epochs
	if c.Branching {
		trainer = abi.TrainBranching
		epochs = abi.BranchEpochs
	}
	model, initial, final, err := trainer(samples, len(contract.Operations), c.Seed)
	info.TrainingNS = time.Since(start).Nanoseconds()
	if err != nil {
		return abi.Artifact{}, nil, nil, info, err
	}
	again, _, _, err := trainer(samples, len(contract.Operations), c.Seed)
	if err != nil {
		return abi.Artifact{}, nil, nil, info, err
	}
	info.BitIdentical = reflect.DeepEqual(model, again)
	info.InitialLoss = initial
	info.FinalLoss = final
	artifact := abi.Artifact{Schema: "phlame.scalar-module.v0", ABI: abi.ABI, Contract: contract, Domain: c.Domain, Model: model, Seed: c.Seed, Epochs: epochs, TrainingHash: info.TrainingHash, VerificationHash: info.VerificationHash, ImplementationHash: implementation, Toolchain: runtime.Version()}
	info.Parameters = 0
	for _, row := range append(append([][]float64{}, model.Hidden...), model.Weights...) {
		info.Parameters += len(row)
	}
	info.BranchCases, info.BranchCorrect = map[string]int{}, map[string]int{}
	affine, _, _, err := abi.Train(samples, len(contract.Operations), c.Seed)
	if err != nil {
		return abi.Artifact{}, nil, nil, info, err
	}
	for _, o := range observed {
		if !o.Accepted {
			continue
		}
		p, err := abi.Decode(model.Predict(len(contract.Operations), o.Operation, o.State, o.Arg))
		correct := err == nil && p == o.Prediction
		if o.Training {
			if correct {
				info.RawTrainingCorrect++
			}
			continue
		}
		scores := model.Predict(len(contract.Operations), o.Operation, o.State, o.Arg)
		info.MaxHeldoutIntegerError = math.Max(info.MaxHeldoutIntegerError, math.Max(math.Abs(scores[0]*abi.Scale-float64(o.Prediction.Next)), math.Abs(scores[1]*abi.Scale-float64(o.Prediction.Result))))
		info.HeldoutCases++
		if correct {
			info.RawHeldoutCorrect++
		}
		label := "read"
		if contract.Operations[o.Operation].Arity == 1 {
			label = "add"
			if c.Branching {
				label = "interior"
			}
			if c.Branching && o.State+o.Arg < -16 {
				label = "lower_saturation"
			}
			if c.Branching && o.State+o.Arg > 16 {
				label = "upper_saturation"
			}
			if c.Branching && (o.State+o.Arg == -16 || o.State+o.Arg == 16) {
				label = "threshold"
			}
		}
		info.BranchCases[label]++
		if correct {
			info.BranchCorrect[label]++
		}
		ap, ae := abi.Decode(affine.Predict(len(contract.Operations), o.Operation, o.State, o.Arg))
		if ae == nil && ap == o.Prediction {
			info.AffineHeldoutCorrect++
		}
	}
	info.ModelKind = "affine"
	if c.Branching {
		info.ModelKind = "residual ReLU, 16 hidden units"
	}
	rawBytes, err := lines(rawPredictions(artifact, observed))
	if err != nil {
		return abi.Artifact{}, nil, nil, info, err
	}
	info.RawPredictionsHash = abi.Hash(rawBytes)
	info.Root = abi.Root(artifact)
	return artifact, samples, observed, info, nil
}
func literal(n int64) *abi.Binding { return &abi.Binding{Literal: &n} }
func prior(i int) *abi.Binding     { return &abi.Binding{From: &i} }
func transactions() []abi.Transaction {
	out := []abi.Transaction{}
	for _, c := range []int64{-16, -8, 0, 8, 16} {
		for _, a := range []int64{-16, -8, 0, 8, 16} {
			for _, delta := range []int64{-4, -1, 0, 1, 4} {
				patterns := [][]abi.Step{
					{{Target: "c", Operation: "increment", Argument: literal(delta)}, {Target: "a", Operation: "add", Argument: prior(0)}, {Target: "c", Operation: "read"}, {Target: "a", Operation: "read"}},
					{{Target: "a", Operation: "add", Argument: literal(delta)}, {Target: "c", Operation: "increment", Argument: prior(0)}, {Target: "a", Operation: "read"}, {Target: "c", Operation: "read"}},
					{{Target: "c", Operation: "increment", Argument: literal(delta)}, {Target: "a", Operation: "add", Argument: prior(0)}, {Target: "c", Operation: "increment", Argument: prior(1)}, {Target: "a", Operation: "read"}, {Target: "c", Operation: "read"}},
					{{Target: "c", Operation: "read"}, {Target: "a", Operation: "add", Argument: prior(0)}, {Target: "c", Operation: "increment", Argument: literal(-delta)}, {Target: "a", Operation: "add", Argument: prior(2)}, {Target: "c", Operation: "read"}, {Target: "a", Operation: "read"}},
					{{Target: "a", Operation: "read"}, {Target: "c", Operation: "increment", Argument: prior(0)}, {Target: "a", Operation: "add", Argument: prior(1)}, {Target: "a", Operation: "read"}, {Target: "c", Operation: "read"}},
					{{Target: "c", Operation: "increment", Argument: literal(delta)}, {Target: "c", Operation: "read"}, {Target: "a", Operation: "add", Argument: prior(1)}, {Target: "a", Operation: "read"}},
				}
				for pattern, steps := range patterns {
					out = append(out, abi.Transaction{ID: fmt.Sprintf("pattern%d:c%d:a%d:d%d", pattern, c, a, delta), Instances: map[string]abi.Instance{"c": {Component: "counter", Value: c, Version: 1}, "a": {Component: "accumulator", Value: a, Version: 1}}, Steps: steps})
				}
			}
		}
	}
	return out
}
func observe(source string, artifacts map[string]abi.Artifact, tx abi.Transaction) (observation, int64, error) {
	return observeCapture(source, artifacts, tx, false)
}
func observeCapture(source string, artifacts map[string]abi.Artifact, tx abi.Transaction, capture bool) (observation, int64, error) {
	o := observation{Transaction: tx, InScope: true, OracleStates: map[string]abi.Instance{}}
	r, err := reference.New(source, reference.Options{CaptureIntegers: true})
	if err != nil {
		return o, 0, err
	}
	handles := map[string]semantics.Value{}
	reverse := map[uint64]string{}
	for _, alias := range []string{"a", "c"} {
		state := tx.Instances[alias]
		h, err := r.Construct(artifacts[state.Component].Contract.Class, semantics.Integer(state.Value))
		if err != nil {
			return o, 0, err
		}
		handles[alias] = h
		reverse[h.Object] = alias
	}
	offset := len(r.Events())
	calls := make([]reference.BoundInvocation, len(tx.Steps))
	for i, step := range tx.Steps {
		call := reference.BoundInvocation{Target: handles[step.Target], Method: step.Operation, RejectFalse: true}
		if step.Argument != nil {
			argument := reference.ValueArgument{From: step.Argument.From}
			if step.Argument.Literal != nil {
				value := semantics.Integer(*step.Argument.Literal)
				argument.Literal = &value
			}
			call.Args = []reference.ValueArgument{argument}
		}
		calls[i] = call
	}
	start := time.Now()
	values, bound, err := r.ComposeBound(calls)
	elapsed := time.Since(start).Nanoseconds()
	o.OracleAccepted = err == nil
	if err != nil {
		var failure *reference.Error
		if !errors.As(err, &failure) || failure.Code != "R_REJECT" {
			return o, 0, err
		}
		o.OracleError = failure.Code
	}
	for _, attempt := range bound {
		alias := reverse[attempt.Target]
		artifact := artifacts[tx.Instances[alias].Component]
		d := artifact.Domain
		state := attempt.Before[artifact.Contract.Slot].Int
		if artifact.Contract.Class == "SaturatingAccumulator" && attempt.Method == "add" {
			if o.ObservedBranches == nil {
				o.ObservedBranches = map[string]int{}
			}
			label := "interior"
			sum := state + attempt.Args[0].Int
			if sum < -16 {
				label = "lower_saturation"
			}
			if sum > 16 {
				label = "upper_saturation"
			}
			if sum == -16 || sum == 16 {
				label = "threshold"
			}
			o.ObservedBranches[label]++
		}
		if state < d.StateMin || state > d.StateMax {
			o.InScope = false
		}
		if len(attempt.Args) > 0 {
			arg := attempt.Args[0].Int
			if arg < d.ArgMin || arg > d.ArgMax {
				o.InScope = false
			}
		}
	}
	for i, v := range values {
		if v.Kind != "int" {
			return o, 0, fmt.Errorf("noninteger cross-component value")
		}
		o.OracleResults = append(o.OracleResults, abi.Return{Type: "int64", Value: v.Int, Producer: tx.Steps[i].Target})
	}
	for _, object := range r.Objects() {
		alias := reverse[object.ID]
		spec := tx.Instances[alias]
		spec.Value = object.Fields[artifacts[spec.Component].Contract.Slot].Int
		spec.Version = object.Version
		o.OracleStates[alias] = spec
	}
	events := r.Events()[offset:]
	if capture {
		o.Events = events
	}
	if o.OracleAccepted {
		for _, event := range events {
			if event.Operation == "stage_write" {
				alias := reverse[event.Object]
				o.OracleWrites = append(o.OracleWrites, abi.Write{Target: alias, Slot: artifacts[tx.Instances[alias].Component].Contract.Slot, Value: event.Integers["value"]})
			}
		}
	}
	raw, _ := json.Marshal(events)
	o.TraceHash = abi.Hash(raw)
	if !o.OracleAccepted {
		o.Category = "native_rejection"
	} else if !o.InScope {
		o.Category = "outside_certified_scope"
	} else {
		o.Category = "accepted"
	}
	if !o.OracleAccepted && !reflect.DeepEqual(o.OracleStates, tx.Instances) {
		return o, 0, fmt.Errorf("oracle rejection leaked cross-component state")
	}
	return o, elapsed, nil
}
func worker(dir, engine, boundary string, cases []abi.Transaction, repeats int) (response, error) {
	input, _ := json.Marshal(struct {
		Transactions []abi.Transaction `json:"transactions"`
		Repeats      int               `json:"repeats"`
	}{cases, repeats})
	command := exec.Command(filepath.Join(dir, "worker"), "registry.json", "trusted-key.hex", engine, boundary)
	command.Dir = dir
	command.Env = []string{"GOMAXPROCS=1"}
	command.Stdin = bytes.NewReader(input)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	raw, err := command.Output()
	if err != nil {
		return response{}, fmt.Errorf("source-free worker: %w: %s", err, &stderr)
	}
	var out response
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	if out.PreflightError == "" && len(out.Outcomes) != len(cases)*repeats {
		return out, fmt.Errorf("result count mismatch")
	}
	return out, nil
}
func summarize(w response, observations []observation) (engineReport, error) {
	s := engineReport{Counts: counts{Scheduled: len(observations)}, Certificates: w.Certificates, PreflightError: w.PreflightError, LoadNS: w.LoadNS, Timing: map[string]metric{}}
	if w.PreflightError != "" {
		if len(w.Outcomes) > 0 {
			return s, fmt.Errorf("failed preflight invoked modules")
		}
		return s, nil
	}
	labels := map[string]observation{}
	for _, o := range observations {
		labels[o.Transaction.ID] = o
	}
	times := map[string][]int64{}
	for i, out := range w.Outcomes {
		o, ok := labels[out.ID]
		if !ok {
			return s, fmt.Errorf("unknown worker transaction")
		}
		abi.AddWork(&s.Work.Relations, out.Work.Relations)
		s.Work.BindingChecks += out.Work.BindingChecks
		s.Work.FrameChecks += out.Work.FrameChecks
		s.Work.DomainChecks += out.Work.DomainChecks
		s.Work.CertificateUses += out.Work.CertificateUses
		abi.AddWork(&s.LookupWork, out.LookupWork)
		for phase, n := range map[string]int64{"proposal": out.Timing.ProposalNS, "binding": out.Timing.BindingNS, "checker": out.Timing.CheckerNS, "commit": out.Timing.CommitNS, "overhead": out.Timing.OverheadNS, "total": out.Timing.TotalNS} {
			times[phase] = append(times[phase], n)
		}
		if out.Accepted && (!o.OracleAccepted || !reflect.DeepEqual(out.States, o.OracleStates) || !reflect.DeepEqual(out.Results, o.OracleResults) || !reflect.DeepEqual(out.Writes, o.OracleWrites)) {
			return s, fmt.Errorf("incorrect accepted transaction %s", out.ID)
		}
		if !out.Accepted && (!reflect.DeepEqual(out.States, o.Transaction.Instances) || len(out.Results) > 0 || len(out.Writes) > 0) {
			return s, fmt.Errorf("rejection leaked state or returns")
		}
		if i >= len(observations) {
			continue
		}
		s.Counts.Evaluated++
		if out.Accepted {
			s.Counts.Accepted++
		} else {
			switch o.Category {
			case "accepted":
				s.Counts.SupportedRejected++
			case "native_rejection":
				s.Counts.NativeRejected++
			default:
				s.Counts.ScopeRejected++
			}
		}
	}
	for phase, v := range times {
		s.Timing[phase] = measure(v)
	}
	return s, nil
}
func run() error {
	branching := flag.Bool("branching", false, "train the saturating accumulator with a ReLU executor")
	output := flag.String("out", "", "new result directory")
	workerPath := flag.String("worker", "bin/phlame-components-worker", "source-free worker")
	repeats := flag.Int("repeats", 10, "timing repetitions")
	flag.Parse()
	if *output == "" {
		*output = "experiments/components/run-0001"
		if *branching {
			*output = "experiments/saturating-accumulator/run-0001"
		}
	}
	if *branching {
		configs = []config{configs[0], branchingConfig()}
	}
	if *repeats < 1 || *repeats > 100 {
		return fmt.Errorf("repeats bounds")
	}
	if _, err := os.Stat(*output); !os.IsNotExist(err) {
		return fmt.Errorf("output must not exist")
	}
	implementation, err := implementationHash()
	if err != nil {
		return err
	}
	r := report{Schema: "phlame.components.experiment.v0", ABI: abi.ABI, ImplementationHash: implementation, Toolchain: runtime.Version(), Platform: runtime.GOOS + "/" + runtime.GOARCH, Repeats: *repeats, Modules: map[string]moduleReport{}, Categories: map[string]int{}, Engines: map[string]engineReport{}, BreakEven: map[string]int64{}, Limitations: []string{
		"Independent accepted-transition models learn affine arithmetic; rejecting branch predicates belong to source-derived contracts, not the models.",
		"A scalar slot per instance, int64 value passing, supplied schedules and finite domains; no learned planning, shared-cell aliases or arbitrary modules.",
		"Module certification stores admission intervals, not transition outputs, and repeats at process startup.",
		"The lookup baseline interprets exact source-derived contracts; it coalesces concrete expected-value computation rather than recomputing it.",
		"Observed source rejections and source-valid but uncertified transactions are reported separately.",
		"Signing uses a public research key; no production tenant permissions, external effect adapters or durable storage exist.",
		"Timing includes instrumentation; workers use GOMAXPROCS=1, without CPU pinning; oracle machinery differs from workers.",
	}}
	if *branching {
		r.Limitations[0] = "A residual ReLU model learns accepted saturating transitions from sparse examples; finite certification proves only the declared integer domain, not real-valued or arbitrary program behavior."
	}
	artifacts := map[string]abi.Artifact{}
	training := map[string][]abi.Sample{}
	primitiveData := map[string][]primitive{}
	combined := ""
	for _, c := range configs {
		artifact, samples, observations, info, err := trainModule(c, implementation)
		if err != nil {
			return err
		}
		artifacts[c.Alias] = artifact
		training[c.Alias] = samples
		primitiveData[c.Alias] = observations
		r.Modules[c.Alias] = info
		source, err := os.ReadFile(c.Source)
		if err != nil {
			return err
		}
		combined += string(source) + "\n"
	}
	oracle, err := reference.New(combined, reference.Options{})
	if err != nil {
		return err
	}
	r.OracleSourceHash = oracle.SourceHash()
	cases := transactions()
	observations := make([]observation, 0, len(cases))
	oracleTimes := []int64{}
	for _, tx := range cases {
		o, elapsed, err := observe(combined, artifacts, tx)
		if err != nil {
			return err
		}
		observations = append(observations, o)
		oracleTimes = append(oracleTimes, elapsed)
		r.Categories[o.Category]++
	}
	r.Cases = len(cases)
	r.Oracle = measure(oracleTimes)
	observationBytes, err := lines(observations)
	if err != nil {
		return err
	}
	r.ObservationHash = abi.Hash(observationBytes)
	dir, err := os.MkdirTemp("", "phlame-modules-source-free-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	signingSeed := sha256.Sum256([]byte("PHLAME independent components public research key v0"))
	key := ed25519.NewKeyFromSeed(signingSeed[:])
	if err := os.WriteFile(filepath.Join(dir, "trusted-key.hex"), []byte(hex.EncodeToString(key.Public().(ed25519.PublicKey))), 0644); err != nil {
		return err
	}
	if err := copyFile(*workerPath, filepath.Join(dir, "worker"), 0755); err != nil {
		return err
	}
	type moduleRef struct {
		File string `json:"file"`
		Root string `json:"module_sha256"`
	}
	manifest := struct {
		ABI     string               `json:"abi"`
		Modules map[string]moduleRef `json:"modules"`
	}{abi.ABI, map[string]moduleRef{}}
	for alias, a := range artifacts {
		name := alias + ".json"
		if err := abi.Freeze(filepath.Join(dir, name), a, key); err != nil {
			return err
		}
		manifest.Modules[alias] = moduleRef{name, abi.Root(a)}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		info := r.Modules[alias]
		info.Bytes = len(data)
		r.Modules[alias] = info
	}
	if err := save(filepath.Join(dir, "registry.json"), manifest); err != nil {
		return err
	}
	binary, err := os.ReadFile(filepath.Join(dir, "worker"))
	if err != nil {
		return err
	}
	r.WorkerHash = abi.Hash(binary)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		r.SourceFreeFiles = append(r.SourceFreeFiles, entry.Name())
	}
	for _, engine := range []string{"learned", "lookup", "zero", "shuffled"} {
		for _, boundary := range []string{"concrete", "certified"} {
			n := *repeats
			if engine == "zero" || engine == "shuffled" {
				n = 1
			}
			w, err := worker(dir, engine, boundary, cases, n)
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
		concrete, certified := r.Engines[engine+"/concrete"], r.Engines[engine+"/certified"]
		cost := int64(0)
		for _, c := range certified.Certificates {
			cost += c.BuildNS
		}
		saving := concrete.Timing["total"].Mean - certified.Timing["total"].Mean
		if saving > 0 {
			r.BreakEven[engine] = int64(math.Ceil(float64(cost) / saving))
		}
	}
	if err := os.MkdirAll(*output, 0755); err != nil {
		return err
	}
	for _, name := range []string{"counter.json", "accumulator.json", "registry.json", "trusted-key.hex"} {
		mode := os.FileMode(0644)
		if name == "counter.json" || name == "accumulator.json" {
			mode = 0444
		}
		if err := copyFile(filepath.Join(dir, name), filepath.Join(*output, name), mode); err != nil {
			return err
		}
	}
	for alias, data := range training {
		rawPredictionsBytes, err := lines(rawPredictions(artifacts[alias], primitiveData[alias]))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(*output, alias+"-predictions.jsonl"), rawPredictionsBytes, 0644); err != nil {
			return err
		}
		if err := save(filepath.Join(*output, alias+"-training.json"), data); err != nil {
			return err
		}
		raw, err := lines(primitiveData[alias])
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(*output, alias+"-primitive.jsonl"), raw, 0644); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(*output, "transactions.jsonl"), observationBytes, 0644); err != nil {
		return err
	}
	if err := save(filepath.Join(*output, "report.json"), r); err != nil {
		return err
	}
	if *branching {
		if err := writeBranchExamples(*output, dir, combined, artifacts); err != nil {
			return err
		}
	}
	fmt.Printf("saved %s: counter train=%d accumulator train=%d transactions=%d categories=%v accepted=%d/%d\n", *output, len(training["counter"]), len(training["accumulator"]), len(cases), r.Categories, r.Engines["learned/concrete"].Counts.Accepted, r.Engines["learned/certified"].Counts.Accepted)
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
