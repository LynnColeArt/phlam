package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LynnColeArt/phlam/runtime/learned"
)

func artifactExecutor(t *testing.T) *learned.Executor {
	t.Helper()
	key, err := learned.PublicKey("../../experiments/counter/run-0001/trusted-key.hex")
	if err != nil {
		t.Fatal(err)
	}
	e, err := learned.Load("../../experiments/counter/run-0001/artifact.json", key)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func TestAllHeldOutCompositionsAgainstOracleAndLookup(t *testing.T) {
	all := sequences()
	if len(all) != 1176 {
		t.Fatal("case count changed", len(all))
	}
	source, err := os.ReadFile("../../examples/counter/counter.phlam")
	if err != nil {
		t.Fatal(err)
	}
	e := artifactExecutor(t)
	plans := map[string]*learned.Composer{}
	for _, engine := range []string{"learned", "lookup"} {
		for _, boundary := range []string{"concrete", "certified"} {
			p, _, err := e.NewComposer(engine, boundary)
			if err != nil {
				t.Fatal(err)
			}
			plans[engine+"/"+boundary] = p
		}
	}
	supported, outside := 0, 0
	seen := map[string]bool{}
	for _, s := range all {
		raw, _ := json.Marshal(struct {
			Initial int64
			Steps   []learned.Operation
		}{s.Initial, s.Steps})
		if seen[string(raw)] {
			t.Fatal("duplicated sequence")
		}
		seen[string(raw)] = true
		if len(s.Steps) < 2 || len(s.Steps) > 16 {
			t.Fatal("not a held-out composition")
		}
		o, _, err := observe(string(source), s)
		if err != nil {
			t.Fatal(err)
		}
		if o.Supported {
			supported++
		} else {
			outside++
		}
		for name, p := range plans {
			outcome := p.Run(s)
			summary, err := summarize(response{Engine: name, Outcomes: []learned.CompositionOutcome{outcome}}, []observation{o})
			if err != nil {
				t.Fatal(err)
			}
			if o.Supported && summary.Counts.Accepted != 1 || !o.Supported && summary.Counts.OutsideRejected != 1 {
				t.Fatalf("coverage mismatch %s %s %+v", name, s.ID, outcome)
			}
		}
	}
	if supported != 1096 || outside != 80 {
		t.Fatal("coverage classification changed", supported, outside)
	}
}
func TestCompositionSourceFreeWorker(t *testing.T) {
	deps, err := exec.Command("go", "list", "-deps", "../phlam-counter-compose-worker").Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(deps), "/runtime/reference") {
		t.Fatal("worker imports oracle")
	}
	dir := t.TempDir()
	command := exec.Command("go", "build", "-o", filepath.Join(dir, "worker"), "../phlam-counter-compose-worker")
	if raw, err := command.CombinedOutput(); err != nil {
		t.Fatal(string(raw), err)
	}
	for _, name := range []string{"artifact.json", "trusted-key.hex"} {
		if err := copyFile("../../experiments/counter/run-0001/"+name, filepath.Join(dir, name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	sequence := learned.Sequence{ID: "demo", Initial: 12, Steps: []learned.Operation{{ID: "Counter.increment", Delta: 3}, {ID: "Counter.read"}, {ID: "Counter.increment", Delta: -7}}}
	w, err := runWorker(dir, "learned", "certified", []learned.Sequence{sequence}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !w.Certificate.Valid || len(w.Outcomes) != 1 || !w.Outcomes[0].Accepted || w.Outcomes[0].State != 8 || w.Outcomes[0].Work.Relation.IntegerPrimitives != 0 {
		t.Fatal("source-free composition failed", w)
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 3 {
		t.Fatal("unexpected source-free files", err)
	}
	invalid, err := runWorker(dir, "zero", "certified", []learned.Sequence{sequence}, 1)
	if err != nil || invalid.PreflightError == "" || len(invalid.Outcomes) != 0 {
		t.Fatal("bad model passed certification", err)
	}
}
