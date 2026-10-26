package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/LynnColeArt/phlam/runtime/bundle"
	abi "github.com/LynnColeArt/phlam/runtime/components"
	"io"
	"os"
	"time"
)

type request struct {
	Transactions []abi.Transaction `json:"transactions"`
	Repeats      int               `json:"repeats"`
}
type entryMeasurement struct {
	ID     string           `json:"id"`
	Repeat int              `json:"repeat"`
	Work   bundle.EntryWork `json:"work"`
	NS     int64            `json:"ns"`
}
type response struct {
	EntryChecks    []entryMeasurement         `json:"entry_checks,omitempty"`
	Schema         string                     `json:"schema"`
	LoadNS         int64                      `json:"load_ns"`
	Certificates   map[string]abi.Certificate `json:"certificates"`
	PreflightError string                     `json:"preflight_error,omitempty"`
	Outcomes       []abi.Outcome              `json:"outcomes,omitempty"`
}

func run() error {
	if len(os.Args) < 3 || os.Args[1] != "run" {
		return fmt.Errorf("usage: phlam-runtime run <bundle> --key <trusted-key.hex> --request <file|-> [--engine learned|lookup] [--boundary certified|concrete] [--phase-clocks=false]")
	}
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	keyPath := flags.String("key", "", "external trusted key")
	input := flags.String("request", "-", "typed request JSON")
	engine := flags.String("engine", "learned", "execution engine")
	boundary := flags.String("boundary", "certified", "exact boundary")
	clocks := flags.Bool("phase-clocks", true, "record phase clocks")
	if err := flags.Parse(os.Args[3:]); err != nil {
		return err
	}
	if *keyPath == "" || flags.NArg() != 0 {
		return fmt.Errorf("explicit --key required")
	}
	key, err := abi.PublicKey(*keyPath)
	if err != nil {
		return err
	}
	start := time.Now()
	m, artifacts, err := bundle.Load(os.Args[2], key)
	if err != nil {
		return err
	}
	out := response{Schema: "phlame.runtime.v0", LoadNS: time.Since(start).Nanoseconds()}
	var reader io.Reader = os.Stdin
	if *input != "-" {
		f, err := os.Open(*input)
		if err != nil {
			return err
		}
		defer f.Close()
		reader = f
	}
	raw, err := io.ReadAll(io.LimitReader(reader, 4*1024*1024+1))
	if err != nil {
		return err
	}
	if len(raw) > 4*1024*1024 {
		return fmt.Errorf("request size limit")
	}
	var r request
	if err := abi.StrictJSON(raw, &r); err != nil {
		return err
	}
	if len(r.Transactions) < 1 || len(r.Transactions) > 10000 || r.Repeats < 1 || r.Repeats > 100 {
		return fmt.Errorf("request bounds")
	}
	registry, certificates, err := abi.NewRegistry(artifacts, *engine, *boundary)
	out.Certificates = certificates
	if err != nil {
		out.PreflightError = err.Error()
		return json.NewEncoder(os.Stdout).Encode(out)
	}
	execute := registry.Run
	if !*clocks {
		execute = registry.RunUninstrumented
	}
	for repeat := 0; repeat < r.Repeats; repeat++ {
		for _, tx := range r.Transactions {
			var start time.Time
			if *clocks {
				start = time.Now()
			}
			work, entryErr := bundle.CheckEntriesMeasured(m, tx)
			measurement := entryMeasurement{ID: tx.ID, Repeat: repeat, Work: work}
			if *clocks {
				measurement.NS = time.Since(start).Nanoseconds()
			}
			out.EntryChecks = append(out.EntryChecks, measurement)
			if err := entryErr; err != nil {
				out.Outcomes = append(out.Outcomes, abi.Outcome{ID: tx.ID, States: copyStates(tx.Instances), FailedStep: -1, Rejection: err.Error()})
				continue
			}
			out.Outcomes = append(out.Outcomes, execute(tx))
		}
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}
func copyStates(in map[string]abi.Instance) map[string]abi.Instance {
	out := map[string]abi.Instance{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
