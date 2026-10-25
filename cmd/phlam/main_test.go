package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommands(t *testing.T) {
	path := "../../examples/counter/counter.phlam"
	var out, stderr bytes.Buffer
	if code := run([]string{"check", path}, &out, &stderr); code != 0 || !strings.Contains(out.String(), "bootstrap syntax checks only") {
		t.Fatalf("check: %d %s %s", code, &out, &stderr)
	}
	out.Reset()
	stderr.Reset()
	if code := run([]string{"inspect", path}, &out, &stderr); code != 0 || !json.Valid(out.Bytes()) {
		t.Fatalf("inspect: %d %s", code, &stderr)
	}
	invalid := filepath.Join(t.TempDir(), "bad.phlam")
	if err := os.WriteFile(invalid, []byte("class Undocumented {}"), 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	stderr.Reset()
	if code := run([]string{"check", invalid}, &out, &stderr); code != 1 || !strings.Contains(stderr.String(), "P101") || !strings.Contains(stderr.String(), "bad.phlam:1:1") {
		t.Fatalf("diagnostic: %d %s", code, &stderr)
	}
	for _, args := range [][]string{nil, {"serve", path}, {"check"}, {"inspect", path, "extra"}} {
		if run(args, &out, &stderr) != 2 {
			t.Fatal("bad usage accepted", args)
		}
	}
	if run([]string{"check", "/nonexistent/phlam-source"}, &out, &stderr) != 1 {
		t.Fatal("missing file accepted")
	}
}

func TestReferenceCLIAndContracts(t *testing.T) {
	var out, stderr bytes.Buffer
	source := "../../examples/counter/counter.phlam"
	if code := run([]string{"contracts", source}, &out, &stderr); code != 0 || !json.Valid(out.Bytes()) || !strings.Contains(out.String(), "ordered_writes") {
		t.Fatalf("contracts: %d %s", code, &stderr)
	}
	out.Reset()
	stderr.Reset()
	if code := run([]string{"run", source, "../../examples/counter/scenario.json"}, &out, &stderr); code != 0 {
		t.Fatalf("run: %d %s", code, &stderr)
	}
	var report executionReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 6 || report.Results[3].Result.Int != 12 || report.Results[4].Result.Int != 12 || report.Results[5].Result.Int != 0 || len(report.Trace) == 0 {
		t.Fatal("wrong scenario output")
	}
	first := out.String()
	out.Reset()
	stderr.Reset()
	if run([]string{"run", source, "../../examples/counter/scenario.json"}, &out, &stderr) != 0 || out.String() != first {
		t.Fatal("nondeterministic execution report")
	}
	out.Reset()
	stderr.Reset()
	if code := run([]string{"run", source, "../../examples/counter/overflow.json"}, &out, &stderr); code != 1 || !strings.Contains(stderr.String(), "R_OVERFLOW") {
		t.Fatalf("overflow: %d %s", code, &stderr)
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Error == "" || len(report.Objects) != 1 || report.Objects[0].Version != 1 {
		t.Fatal("failure report lost rollback evidence")
	}
	invalid := filepath.Join(t.TempDir(), "invalid.json")
	for _, data := range []string{`{"schema":"phlam.scenario.v0","steps":[],"unknown":1}`, `{} {}`, `{"schema":"phlam.scenario.v0","steps":[{"op":"call","target":"missing","method":"read"}]}`} {
		if err := os.WriteFile(invalid, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		out.Reset()
		stderr.Reset()
		if run([]string{"run", source, invalid}, &out, &stderr) != 1 {
			t.Fatal("invalid scenario accepted")
		}
	}
}
