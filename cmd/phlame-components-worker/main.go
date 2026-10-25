// This worker contains neither source parsing nor an oracle import. The generic
// registry binds independently frozen modules to instance-owned scalar frames.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/LynnColeArt/phlam/runtime/components"
)

type moduleFile struct {
	File string `json:"file"`
	Root string `json:"module_sha256"`
}
type manifest struct {
	ABI     string                `json:"abi"`
	Modules map[string]moduleFile `json:"modules"`
}
type request struct {
	Transactions []components.Transaction `json:"transactions"`
	Repeats      int                      `json:"repeats"`
}
type response struct {
	Schema         string                            `json:"schema"`
	LoadNS         int64                             `json:"load_ns"`
	Certificates   map[string]components.Certificate `json:"certificates"`
	PreflightError string                            `json:"preflight_error,omitempty"`
	Outcomes       []components.Outcome              `json:"outcomes,omitempty"`
}

func run() error {
	if len(os.Args) != 5 {
		return fmt.Errorf("usage: phlame-components-worker <registry.json> <public-key.hex> <engine> <concrete|certified>")
	}
	key, err := components.PublicKey(os.Args[2])
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		return err
	}
	if len(raw) > 16384 {
		return fmt.Errorf("registry manifest size limit")
	}
	var m manifest
	if err := components.StrictJSON(raw, &m); err != nil {
		return err
	}
	if m.ABI != components.ABI || len(m.Modules) == 0 || len(m.Modules) > 8 {
		return fmt.Errorf("invalid registry manifest")
	}
	artifacts := map[string]components.Artifact{}
	start := time.Now()
	for alias, ref := range m.Modules {
		if alias == "" || filepath.Base(ref.File) != ref.File {
			return fmt.Errorf("invalid module file binding")
		}
		a, root, err := components.Load(filepath.Join(filepath.Dir(os.Args[1]), ref.File), key)
		if err != nil {
			return err
		}
		if root != ref.Root {
			return fmt.Errorf("module root mismatch")
		}
		artifacts[alias] = a
	}
	out := response{Schema: "phlame.components.worker.v0", LoadNS: time.Since(start).Nanoseconds()}
	raw, err = io.ReadAll(io.LimitReader(os.Stdin, 4*1024*1024+1))
	if err != nil {
		return err
	}
	if len(raw) > 4*1024*1024 {
		return fmt.Errorf("request size limit")
	}
	var in request
	if err := components.StrictJSON(raw, &in); err != nil {
		return err
	}
	if len(in.Transactions) == 0 || len(in.Transactions) > 10000 || in.Repeats < 1 || in.Repeats > 100 {
		return fmt.Errorf("request bounds")
	}
	registry, certificates, err := components.NewRegistry(artifacts, os.Args[3], os.Args[4])
	out.Certificates = certificates
	if err != nil {
		out.PreflightError = err.Error()
		return json.NewEncoder(os.Stdout).Encode(out)
	}
	out.Outcomes = make([]components.Outcome, 0, len(in.Transactions)*in.Repeats)
	for repeat := 0; repeat < in.Repeats; repeat++ {
		for _, tx := range in.Transactions {
			out.Outcomes = append(out.Outcomes, registry.Run(tx))
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
