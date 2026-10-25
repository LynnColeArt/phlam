package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/LynnColeArt/phlam/compiler/frontend"
	"io"
	"os"
)

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 || args[0] != "check" && args[0] != "inspect" {
		fmt.Fprintln(stderr, "usage: phlam <check|inspect> <source.phlam>")
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
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(program); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
