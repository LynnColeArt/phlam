package main

import (
	"github.com/LynnColeArt/phlam/runtime/learned"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExhaustiveDisjointNumericSplits(t *testing.T) {
	all := cases()
	seen := map[string]bool{}
	train, test := 0, 0
	for _, c := range all {
		if seen[c.ID] {
			t.Fatal("duplicate or leaking split", c.ID)
		}
		seen[c.ID] = true
		if c.Split == "train" {
			train++
		} else {
			test++
		}
	}
	if train != 170 || test != 4216 || len(all) != 129*34 {
		t.Fatal("domain coverage changed", train, test, len(all))
	}
	source, err := os.ReadFile("../../examples/counter/counter.phlam")
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{0, 100, len(all) - 1} {
		label, _, err := observe(string(source), all[index])
		if err != nil || len(label.TraceHash) != 64 {
			t.Fatal("missing observed label provenance", err)
		}
	}
}

func TestSourceFreeWorkerProcess(t *testing.T) {
	dependencies, err := exec.Command("go", "list", "-deps", "../phlam-counter-worker").Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dependencies), "/runtime/reference") {
		t.Fatal("worker imports oracle")
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "worker")
	command := exec.Command("go", "build", "-o", binary, "../phlam-counter-worker")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build worker: %s %v", output, err)
	}
	const folder = "../../experiments/counter/run-0001/"
	for _, name := range []string{"artifact.json", "trusted-key.hex"} {
		if err := copyFile(folder+name, filepath.Join(dir, name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	cases := []learned.Case{{ID: "increment", Operation: "Counter.increment", State: 63, Delta: 15}, {ID: "read", Operation: "Counter.read", State: -63}}
	response, err := worker(dir, "learned", cases, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !response.Outcomes[0].Accepted || response.Outcomes[0].Result != 78 || !response.Outcomes[1].Accepted || response.Outcomes[1].Result != -63 {
		t.Fatal("source-free inference failed")
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 3 {
		t.Fatal("unexpected source-free contents", err)
	}
}
