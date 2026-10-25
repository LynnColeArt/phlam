package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/LynnColeArt/phlam/runtime/learned"
)

type request struct {
	Sequences []learned.Sequence `json:"sequences"`
	Repeats   int                `json:"repeats"`
}
type response struct {
	Schema         string                       `json:"schema"`
	Engine         string                       `json:"engine"`
	Boundary       string                       `json:"boundary"`
	LoadNS         int64                        `json:"load_ns"`
	Certificate    learned.Certificate          `json:"certificate"`
	PreflightError string                       `json:"preflight_error,omitempty"`
	Outcomes       []learned.CompositionOutcome `json:"outcomes"`
}

func run() error {
	if len(os.Args) != 5 {
		return fmt.Errorf("usage: phlam-counter-compose-worker <artifact> <trusted-key> <engine> <concrete|certified>")
	}
	key, err := learned.PublicKey(os.Args[2])
	if err != nil {
		return err
	}
	start := time.Now()
	e, err := learned.Load(os.Args[1], key)
	if err != nil {
		return err
	}
	load := time.Since(start).Nanoseconds()
	data, err := io.ReadAll(io.LimitReader(os.Stdin, 4*1024*1024+1))
	if err != nil {
		return err
	}
	if len(data) > 4*1024*1024 {
		return fmt.Errorf("request size limit")
	}
	var in request
	if err := learned.StrictJSON(data, &in); err != nil {
		return err
	}
	if len(in.Sequences) == 0 || len(in.Sequences) > 10000 || in.Repeats < 1 || in.Repeats > 100 {
		return fmt.Errorf("request bounds invalid")
	}
	p, certificate, err := e.NewComposer(os.Args[3], os.Args[4])
	out := response{Schema: "phlam.counter.compositions.worker.v0", Engine: os.Args[3], Boundary: os.Args[4], LoadNS: load, Certificate: certificate}
	if err != nil {
		out.PreflightError = err.Error()
		return json.NewEncoder(os.Stdout).Encode(out)
	}
	out.Outcomes = make([]learned.CompositionOutcome, 0, len(in.Sequences)*in.Repeats)
	for repeat := 0; repeat < in.Repeats; repeat++ {
		for _, s := range in.Sequences {
			out.Outcomes = append(out.Outcomes, p.Run(s))
		}
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
