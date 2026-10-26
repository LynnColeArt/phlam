package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceFreeProfileModesAndOracleValidation(t *testing.T) {
	deps, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"/compiler/", "/runtime/reference", "/runtime/learned"} {
		if strings.Contains(string(deps), forbidden) {
			t.Fatal("profiling process imports source machinery", forbidden)
		}
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "worker")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatal(string(out), err)
	}
	study := "../../experiments/saturating-accumulator/run-0001"
	for _, name := range []string{"registry.json", "trusted-key.hex", "counter.json", "accumulator.json", "transactions.jsonl"} {
		data, err := os.ReadFile(filepath.Join(study, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	hashes := map[string]bool{}
	for _, engine := range []string{"learned", "lookup"} {
		for _, boundary := range []string{"concrete", "certified"} {
			for _, timed := range []string{"true", "false"} {
				for _, group := range []string{"mixed", "accepted", "scope_rejected"} {
					cmd := exec.Command(binary, "-engine", engine, "-boundary", boundary, "-phase-clocks="+timed, "-group", group, "-iterations", "10")
					cmd.Dir = dir
					raw, err := cmd.CombinedOutput()
					if err != nil {
						t.Fatal(string(raw), err)
					}
					var out output
					if err := json.Unmarshal(raw, &out); err != nil {
						t.Fatal(err)
					}
					if out.VerifiedCases != 750 || out.VerifiedAccepted != 224 || out.VerifiedRejected != 526 {
						t.Fatal("oracle coverage changed", out)
					}
					want := 750
					if group == "accepted" {
						want = 224
					}
					if group == "scope_rejected" {
						want = 526
					}
					if out.GroupCases != want {
						t.Fatal("wrong workload grouping", out)
					}
					if boundary == "certified" && out.Work.Relations.Arithmetic != 0 {
						t.Fatal("warm checker did application arithmetic")
					}
					if timed == "false" && out.Phases.TotalNS != 0 || timed == "true" && out.Phases.TotalNS <= 0 {
						t.Fatal("clock option ignored")
					}
					hashes[out.BehaviorHash] = true
				}
			}
		}
	}
	if len(hashes) != 1 {
		t.Fatal("engines/boundaries/clocks disagree on behavior", hashes)
	}
	// Also exercise the explicit two-component saturation-then-failure example
	// in the unclocked source-free process, outside the 750-case benchmark mix.
	exampleData, err := os.ReadFile(filepath.Join(study, "oracle-traces.json"))
	if err != nil {
		t.Fatal(err)
	}
	var examples []json.RawMessage
	if err := json.Unmarshal(exampleData, &examples); err != nil {
		t.Fatal(err)
	}
	var exampleLines strings.Builder
	for _, row := range examples {
		var compact any
		if err := json.Unmarshal(row, &compact); err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(compact)
		if err != nil {
			t.Fatal(err)
		}
		exampleLines.Write(raw)
		exampleLines.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(dir, "transactions.jsonl"), []byte(exampleLines.String()), 0644); err != nil {
		t.Fatal(err)
	}
	for _, engine := range []string{"learned", "lookup"} {
		cmd := exec.Command(binary, "-engine", engine, "-iterations", "4")
		cmd.Dir = dir
		raw, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(string(raw), err)
		}
		var out output
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatal(err)
		}
		if out.VerifiedCases != 4 || out.VerifiedAccepted != 3 || out.VerifiedRejected != 1 {
			t.Fatal("saturation rollback changed", out)
		}
	}
	// Corrupting oracle evidence must stop the run, not turn into a speed result.
	data, err := os.ReadFile(filepath.Join(dir, "transactions.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"category":"accepted"`, `"category":"outside_certified_scope"`, 1))
	if err := os.WriteFile(filepath.Join(dir, "transactions.jsonl"), data, 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, "-iterations", "10")
	cmd.Dir = dir
	if err := cmd.Run(); err == nil {
		t.Fatal("incorrect oracle evidence accepted")
	}
}
