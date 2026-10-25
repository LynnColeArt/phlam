package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	abi "github.com/LynnColeArt/phlam/runtime/components"
)

func studies(t *testing.T) (map[string]abi.Artifact, string) {
	t.Helper()
	artifacts := map[string]abi.Artifact{}
	combined := ""
	for _, c := range configs {
		c.Source = "../../" + c.Source
		a, _, _, info, err := trainModule(c, abi.Hash([]byte("test implementation")))
		if err != nil {
			t.Fatal(err)
		}
		if !info.BitIdentical || info.FinalLoss > 1e-12 {
			t.Fatal("training failure", info)
		}
		want := 30
		if c.Alias == "accumulator" {
			want = 48
		}
		if info.TrainingSamples != want || info.Parameters != 15 {
			t.Fatal("unexpected independent training task", info)
		}
		artifacts[c.Alias] = a
		source, err := os.ReadFile(c.Source)
		if err != nil {
			t.Fatal(err)
		}
		combined += string(source) + "\n"
	}
	return artifacts, combined
}
func TestAllCrossComponentTransactions(t *testing.T) {
	artifacts, source := studies(t)
	plans := map[string]*abi.Registry{}
	for _, engine := range []string{"learned", "lookup"} {
		for _, boundary := range []string{"concrete", "certified"} {
			r, _, err := abi.NewRegistry(artifacts, engine, boundary)
			if err != nil {
				t.Fatal(err)
			}
			plans[engine+"/"+boundary] = r
		}
	}
	counts := map[string]int{}
	cases := transactions()
	if len(cases) != 750 {
		t.Fatal("transaction coverage changed")
	}
	for _, tx := range cases {
		o, _, err := observe(source, artifacts, tx)
		if err != nil {
			t.Fatal(err)
		}
		counts[o.Category]++
		for name, r := range plans {
			out := r.Run(tx)
			_, err := summarize(response{Outcomes: []abi.Outcome{out}}, []observation{o})
			if err != nil {
				t.Fatal(name, err)
			}
			if o.Category == "accepted" && !out.Accepted || o.Category != "accepted" && out.Accepted {
				t.Fatalf("coverage mismatch %s %s %+v", name, tx.ID, out)
			}
		}
	}
	if !reflect.DeepEqual(counts, map[string]int{"accepted": 188, "native_rejection": 416, "outside_certified_scope": 146}) {
		t.Fatal("coverage classification changed", counts)
	}
}
func TestIndependentTrainingAndReplacementCertificates(t *testing.T) {
	artifacts, _ := studies(t)
	_, original, err := abi.NewRegistry(artifacts, "learned", "certified")
	if err != nil {
		t.Fatal(err)
	}
	_, alone, err := abi.NewRegistry(map[string]abi.Artifact{"counter": artifacts["counter"]}, "learned", "certified")
	if err != nil || alone["counter"].Key != original["counter"].Key {
		t.Fatal("counter proof depends on peer", err)
	}
	cfg := configs[0]
	cfg.Source = "../../" + cfg.Source
	cfg.Seed = 303
	replacement, _, _, _, err := trainModule(cfg, abi.Hash([]byte("test implementation")))
	if err != nil {
		t.Fatal(err)
	}
	_, changed, err := abi.NewRegistry(map[string]abi.Artifact{"counter": replacement, "accumulator": artifacts["accumulator"]}, "learned", "certified")
	if err != nil || changed["counter"].Key == original["counter"].Key || changed["accumulator"].Key != original["accumulator"].Key {
		t.Fatal("replacement polluted peer certificate", err)
	}
	cfg.Seed = configs[0].Seed
	same, _, _, _, err := trainModule(cfg, abi.Hash([]byte("test implementation")))
	if err != nil || abi.Root(same) != abi.Root(artifacts["counter"]) {
		t.Fatal("independent training changed with execution order", err)
	}
}
func TestSourceFreeGenericWorker(t *testing.T) {
	artifacts, _ := studies(t)
	deps, err := exec.Command("go", "list", "-deps", "../phlame-components-worker").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"/runtime/reference", "/compiler/", "/runtime/learned"} {
		if strings.Contains(string(deps), forbidden) {
			t.Fatal("worker depends on component-specific or source machinery", forbidden)
		}
	}
	dir := t.TempDir()
	command := exec.Command("go", "build", "-o", filepath.Join(dir, "worker"), "../phlame-components-worker")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatal(string(output), err)
	}
	seed := sha256.Sum256([]byte("public test signing key"))
	key := ed25519.NewKeyFromSeed(seed[:])
	if err := os.WriteFile(filepath.Join(dir, "trusted-key.hex"), []byte(hex.EncodeToString(key.Public().(ed25519.PublicKey))), 0644); err != nil {
		t.Fatal(err)
	}
	type moduleRef struct {
		File string `json:"file"`
		Root string `json:"module_sha256"`
	}
	manifest := struct {
		ABI     string               `json:"abi"`
		Modules map[string]moduleRef `json:"modules"`
	}{abi.ABI, map[string]moduleRef{}}
	for alias, a := range artifacts {
		file := alias + ".json"
		if err := abi.Freeze(filepath.Join(dir, file), a, key); err != nil {
			t.Fatal(err)
		}
		manifest.Modules[alias] = moduleRef{file, abi.Root(a)}
	}
	if err := save(filepath.Join(dir, "registry.json"), manifest); err != nil {
		t.Fatal(err)
	}
	success := abi.Transaction{ID: "success", Instances: map[string]abi.Instance{"c": {Component: "counter", Value: 2, Version: 1}, "a": {Component: "accumulator", Value: 0, Version: 1}}, Steps: []abi.Step{{Target: "c", Operation: "increment", Argument: literal(1)}, {Target: "a", Operation: "add", Argument: prior(0)}, {Target: "c", Operation: "read"}, {Target: "a", Operation: "read"}}}
	failure := success
	failure.ID = "rollback"
	failure.Instances = map[string]abi.Instance{"c": {Component: "counter", Value: 8, Version: 1}, "a": {Component: "accumulator", Value: 0, Version: 1}}
	out, err := worker(dir, "learned", "certified", []abi.Transaction{success, failure}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Outcomes[0].Accepted || out.Outcomes[0].States["c"].Value != 3 || out.Outcomes[0].States["a"].Value != 3 || out.Outcomes[1].Accepted || !reflect.DeepEqual(out.Outcomes[1].States, failure.Instances) {
		t.Fatal("source-free modules failed", out)
	}
	if out.Outcomes[0].Work.Relations.Arithmetic != 0 || out.Outcomes[0].Results[1].Producer != "a" {
		t.Fatal("wrong ABI or checker work")
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 5 {
		t.Fatal("source-free worker has unexpected files", err)
	}
	// A forged value type cannot enter the typed register file.
	malformed := exec.Command(filepath.Join(dir, "worker"), "registry.json", "trusted-key.hex", "learned", "certified")
	malformed.Dir = dir
	malformed.Stdin = strings.NewReader(`{"transactions":[{"id":"bad-type","instances":{"c":{"component":"counter","value":0,"version":1}},"steps":[{"target":"c","operation":"increment","argument":{"literal":"not-an-integer"}}]}],"repeats":1}`)
	if err := malformed.Run(); err == nil {
		t.Fatal("forged value type accepted")
	}
	// Root pinning rejects a module swapped under the same registry alias.
	wrong := manifest.Modules["counter"]
	wrong.Root = manifest.Modules["accumulator"].Root
	manifest.Modules["counter"] = wrong
	if err := save(filepath.Join(dir, "registry.json"), manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := worker(dir, "learned", "certified", []abi.Transaction{success}, 1); err == nil {
		t.Fatal("registry accepted a swapped module root")
	}
}
