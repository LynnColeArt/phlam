package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/LynnColeArt/phlam/runtime/reference"
	"github.com/LynnColeArt/phlam/runtime/trace"
	"github.com/LynnColeArt/phlam/semantics"
)

type scenario struct {
	Schema          string         `json:"schema"`
	CaptureIntegers bool           `json:"capture_integers"`
	Steps           []scenarioStep `json:"steps"`
}
type scenarioStep struct {
	Op     string  `json:"op"`
	Bind   string  `json:"bind,omitempty"`
	Class  string  `json:"class,omitempty"`
	From   string  `json:"from,omitempty"`
	Target string  `json:"target,omitempty"`
	Method string  `json:"method,omitempty"`
	Args   []int64 `json:"args,omitempty"`
}
type scenarioResult struct {
	Step   int             `json:"step"`
	Result semantics.Value `json:"result"`
}
type executionReport struct {
	Schema     string                     `json:"schema"`
	Backend    string                     `json:"backend"`
	SourceHash string                     `json:"source_sha256"`
	Results    []scenarioResult           `json:"results"`
	Bindings   map[string]semantics.Value `json:"bindings"`
	Objects    []reference.Object         `json:"objects"`
	Trace      []trace.Event              `json:"trace"`
	Error      string                     `json:"error,omitempty"`
}

func runScenario(source, path string) (executionReport, error) {
	report := executionReport{Schema: "phlam.reference-run.v0", Backend: "reference", Results: []scenarioResult{}, Bindings: map[string]semantics.Value{}}
	file, err := os.Open(path)
	if err != nil {
		return report, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var spec scenario
	if err := decoder.Decode(&spec); err != nil {
		return report, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return report, fmt.Errorf("scenario must contain exactly one JSON document")
	}
	if spec.Schema != "phlam.scenario.v0" || len(spec.Steps) == 0 || len(spec.Steps) > 1000 {
		return report, fmt.Errorf("scenario requires phlam.scenario.v0 and 1..1000 steps")
	}
	r, err := reference.New(source, reference.Options{CaptureIntegers: spec.CaptureIntegers})
	if err != nil {
		return report, err
	}
	report.SourceHash = r.SourceHash()
	for i, step := range spec.Steps {
		values := make([]semantics.Value, len(step.Args))
		for j, n := range step.Args {
			values[j] = semantics.Integer(n)
		}
		result := semantics.Null()
		switch step.Op {
		case "new":
			if step.Bind == "" || step.Class == "" || step.Target != "" || step.From != "" || step.Method != "" {
				err = fmt.Errorf("invalid new step")
				break
			}
			if _, ok := report.Bindings[step.Bind]; ok {
				err = fmt.Errorf("binding already exists")
				break
			}
			result, err = r.Construct(step.Class, values...)
			if err == nil {
				report.Bindings[step.Bind] = result
			}
		case "alias":
			if step.Bind == "" || step.From == "" || step.Class != "" || step.Target != "" || step.Method != "" || len(step.Args) > 0 {
				err = fmt.Errorf("invalid alias step")
				break
			}
			if _, ok := report.Bindings[step.Bind]; ok {
				err = fmt.Errorf("binding already exists")
				break
			}
			var ok bool
			result, ok = report.Bindings[step.From]
			if !ok {
				err = fmt.Errorf("unknown alias source")
				break
			}
			report.Bindings[step.Bind] = result
		case "call":
			if step.Target == "" || step.Method == "" || step.Bind != "" || step.Class != "" || step.From != "" {
				err = fmt.Errorf("invalid call step")
				break
			}
			target, ok := report.Bindings[step.Target]
			if !ok {
				err = fmt.Errorf("unknown target binding")
				break
			}
			result, err = r.Call(target, step.Method, values...)
		default:
			err = fmt.Errorf("unknown scenario operation %q", step.Op)
		}
		if err != nil {
			report.Error = err.Error()
			break
		}
		report.Results = append(report.Results, scenarioResult{Step: i, Result: result})
	}
	report.Objects = r.Objects()
	report.Trace = r.Events()
	return report, err
}
