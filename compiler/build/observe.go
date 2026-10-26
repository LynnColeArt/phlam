package build

import (
	"context"
	"encoding/json"
	"fmt"
	compile "github.com/LynnColeArt/phlam/compiler/components"
	"github.com/LynnColeArt/phlam/compiler/frontend"
	abi "github.com/LynnColeArt/phlam/runtime/components"
	"github.com/LynnColeArt/phlam/runtime/reference"
	"github.com/LynnColeArt/phlam/semantics"
	"time"
)

func observe(ctx context.Context, c Component, p Plan, source string) (abi.Contract, []abi.Sample, []Observation, int64, error) {
	start := time.Now()
	contract, err := compile.Compile(source, c.Class)
	if err != nil {
		return contract, nil, nil, 0, sourceError(c.Source, source, err)
	}
	ast, err := frontend.Parse(source)
	if err != nil {
		return contract, nil, nil, 0, sourceError(c.Source, source, err)
	}
	if len(ast.Classes) != 1 {
		return contract, nil, nil, 0, fmt.Errorf("%s:1:1: G01 requires one explicit class per source", c.Source)
	}
	names := map[string]bool{}
	for _, op := range contract.Operations {
		names[op.Name] = true
	}
	entries := map[string]bool{}
	for _, entry := range c.Entries {
		if !names[entry] || entries[entry] {
			return contract, nil, nil, 0, fmt.Errorf("%s: unknown/duplicate entry %s", c.Source, entry)
		}
		entries[entry] = true
	}
	cases := 0
	for _, op := range contract.Operations {
		args := int(c.Domain.ArgMax - c.Domain.ArgMin + 1)
		if op.Arity == 0 {
			args = 1
		}
		cases += int(c.Domain.StateMax-c.Domain.StateMin+1) * args
	}
	if cases > p.MaxCases {
		return contract, nil, nil, 0, fmt.Errorf("%s: primitive case budget exceeded", c.Source)
	}
	samples := []abi.Sample{}
	observations := []Observation{}
	for index, op := range contract.Operations {
		for state := c.Domain.StateMin; state <= c.Domain.StateMax; state++ {
			lo, hi := c.Domain.ArgMin, c.Domain.ArgMax
			if op.Arity == 0 {
				lo, hi = 0, 0
			}
			for arg := lo; arg <= hi; arg++ {
				if err := ctx.Err(); err != nil {
					return contract, nil, nil, 0, err
				}
				oracle, err := reference.New(source, reference.Options{CaptureIntegers: true})
				if err != nil {
					return contract, nil, nil, 0, sourceError(c.Source, source, err)
				}
				handle, err := oracle.Construct(c.Class, semantics.Integer(state))
				if err != nil {
					return contract, nil, nil, 0, sourceError(c.Source, source, err)
				}
				args := []semantics.Value{}
				if op.Arity == 1 {
					args = append(args, semantics.Integer(arg))
				}
				value, err := oracle.Call(handle, op.Name, args...)
				if err != nil {
					return contract, nil, nil, 0, sourceError(c.Source, source, err)
				}
				object := oracle.Objects()[0]
				accepted := value.Kind == "int"
				prediction := abi.Prediction{Next: object.Fields[contract.Slot].Int, Result: value.Int, Write: object.Version == 2}
				var work abi.Work
				expected, err := op.Evaluate(state, arg, &work)
				if err != nil || !accepted && value != semantics.Boolean(false) || expected.Accepted != accepted || accepted && expected.Prediction != prediction || !accepted && (object.Version != 1 || prediction.Next != state) {
					return contract, nil, nil, 0, fmt.Errorf("%s: source/contract mismatch in %s at state=%d arg=%d", c.Source, op.Name, state, arg)
				}
				trace, _ := json.Marshal(oracle.Events())
				training := accepted && state%p.Sampling.StateStride == 0 && (op.Arity == 0 || arg%p.Sampling.ArgStride == 0)
				row := Observation{Operation: index, State: state, Arg: arg, Accepted: accepted, Prediction: prediction, Training: training, TraceHash: abi.Hash(trace)}
				if state == c.Domain.StateMin && arg == lo {
					row.Trace = oracle.Events()
				}
				observations = append(observations, row)
				if training {
					samples = append(samples, abi.Sample{Operation: index, State: state, Arg: arg, Prediction: prediction, TraceHash: abi.Hash(trace)})
				}
			}
		}
	}
	if len(samples) == 0 {
		return contract, nil, nil, 0, fmt.Errorf("%s: no admitted training samples", c.Source)
	}
	return contract, samples, observations, time.Since(start).Nanoseconds(), nil
}
func sourceError(file, source string, err error) error {
	if d, ok := err.(*frontend.Diagnostic); ok {
		return fmt.Errorf("%s", d.Render(file, source))
	}
	if e, ok := err.(*reference.Error); ok {
		return fmt.Errorf("%s:%d:%d: %s: %s", file, e.Span.Line, e.Span.Column, e.Code, e.Message)
	}
	return fmt.Errorf("%s:1:1: %w", file, err)
}
