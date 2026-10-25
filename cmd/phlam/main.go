package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/LynnColeArt/phlam/compiler/contracts"
	"github.com/LynnColeArt/phlam/compiler/frontend"
	"github.com/LynnColeArt/phlam/runtime/reference"
)

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 || (args[0] == "run" && len(args) != 3) || (args[0] != "run" && (len(args) != 2 || args[0] != "check" && args[0] != "inspect" && args[0] != "contracts")) {
		fmt.Fprintln(stderr, "usage: phlam <check|inspect|contracts> <source.phlam> | phlam run <source.phlam> <scenario.json>")
		return 2
	}
	data, err := os.ReadFile(args[1])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	program, err := frontend.Parse(string(data))
	if err != nil {
		var d *frontend.Diagnostic
		if errors.As(err, &d) {
			fmt.Fprint(stderr, d.Render(args[1], string(data)))
		} else {
			fmt.Fprintln(stderr, err)
		}
		return 1
	}
	if args[0] == "check" {
		fmt.Fprintf(stdout, "ok: %s (%d classes; bootstrap syntax checks only)\n", args[1], len(program.Classes))
		return 0
	}
	var output any = program
	if args[0] == "contracts" {
		output = contracts.Derive(program)
	}
	var executionErr error
	if args[0] == "run" {
		output, executionErr = runScenario(string(data), args[2])
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if executionErr != nil {
		var runtimeErr *reference.Error
		if errors.As(executionErr, &runtimeErr) {
			diagnostic := frontend.Diagnostic{Code: runtimeErr.Code, Message: runtimeErr.Message, Hint: "inspect the execution trace; failed transactions publish no state", Span: runtimeErr.Span}
			fmt.Fprint(stderr, diagnostic.Render(args[1], string(data)))
			for _, frame := range runtimeErr.Stack {
				fmt.Fprintf(stderr, " in %s at %d:%d (call %d)\n", frame.Operation, frame.Span.Line, frame.Span.Column, frame.Invocation)
			}
		} else {
			fmt.Fprintln(stderr, executionErr)
		}
		return 1
	}
	return 0
}
func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
