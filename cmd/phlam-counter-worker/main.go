// The worker imports no reference oracle and accepts no source path. Its only
// executable application input is the signed learned artifact plus typed cases.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/LynnColeArt/phlam/runtime/learned"
)

type Request struct {
	Cases   []learned.Case `json:"cases"`
	Repeats int            `json:"repeats"`
}
type Response struct {
	Schema         string            `json:"schema"`
	Engine         string            `json:"engine"`
	LoadNS         int64             `json:"load_ns"`
	AllocatedBytes uint64            `json:"allocated_bytes"`
	Outcomes       []learned.Outcome `json:"outcomes"`
}

func run() error {
	if len(os.Args) != 4 {
		return fmt.Errorf("usage: phlam-counter-worker <artifact.json> <trusted-key.hex> <learned|lookup|zero|shuffled>")
	}
	engine := os.Args[3]
	if engine != "learned" && engine != "lookup" && engine != "zero" && engine != "shuffled" {
		return fmt.Errorf("unknown engine")
	}
	key, err := learned.PublicKey(os.Args[2])
	if err != nil {
		return err
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	start := time.Now()
	e, err := learned.Load(os.Args[1], key)
	if err != nil {
		return err
	}
	load := time.Since(start).Nanoseconds()
	runtime.ReadMemStats(&after)
	data, err := io.ReadAll(io.LimitReader(os.Stdin, 4*1024*1024+1))
	if err != nil {
		return err
	}
	if len(data) > 4*1024*1024 {
		return fmt.Errorf("request size limit")
	}
	var request Request
	if err := learned.StrictJSON(data, &request); err != nil {
		return err
	}
	if len(request.Cases) == 0 || len(request.Cases) > 10000 || request.Repeats < 1 || request.Repeats > 100 {
		return fmt.Errorf("request bounds invalid")
	}
	out := Response{Schema: "phlam.counter.worker.v0", Engine: engine, LoadNS: load, AllocatedBytes: after.TotalAlloc - before.TotalAlloc, Outcomes: make([]learned.Outcome, 0, len(request.Cases)*request.Repeats)}
	for i := 0; i < request.Repeats; i++ {
		for _, c := range request.Cases {
			out.Outcomes = append(out.Outcomes, e.Run(c, engine))
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
