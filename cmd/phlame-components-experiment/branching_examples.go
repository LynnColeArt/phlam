package main

import (
	abi "github.com/LynnColeArt/phlam/runtime/components"
	"path/filepath"
)

func writeBranchExamples(output, dir, source string, artifacts map[string]abi.Artifact) error {
	cases := []abi.Transaction{}
	for _, c := range []struct {
		id          string
		c, a, delta int64
	}{
		{"upper", 2, 15, 3}, {"lower", -2, -15, -3}, {"threshold", 1, 15, 0}, {"saturate-then-rollback", 0, 16, 1},
	} {
		steps := []abi.Step{{Target: "c", Operation: "increment", Argument: literal(c.delta)}, {Target: "a", Operation: "add", Argument: prior(0)}}
		if c.id == "saturate-then-rollback" {
			steps = append(steps, abi.Step{Target: "c", Operation: "increment", Argument: prior(1)})
		} else {
			steps = append(steps, abi.Step{Target: "c", Operation: "read"}, abi.Step{Target: "a", Operation: "read"})
		}
		cases = append(cases, abi.Transaction{ID: c.id, Instances: map[string]abi.Instance{"c": {Component: "counter", Value: c.c, Version: 1}, "a": {Component: "accumulator", Value: c.a, Version: 1}}, Steps: steps})
	}
	observations := []observation{}
	for _, tx := range cases {
		o, _, err := observeCapture(source, artifacts, tx, true)
		if err != nil {
			return err
		}
		observations = append(observations, o)
	}
	if err := save(filepath.Join(output, "request.json"), struct {
		Transactions []abi.Transaction `json:"transactions"`
		Repeats      int               `json:"repeats"`
	}{cases, 1}); err != nil {
		return err
	}
	if err := save(filepath.Join(output, "oracle-traces.json"), observations); err != nil {
		return err
	}
	w, err := worker(dir, "learned", "certified", cases, 1)
	if err != nil {
		return err
	}
	if _, err := summarize(w, observations); err != nil {
		return err
	}
	return save(filepath.Join(output, "example-results.json"), w)
}
