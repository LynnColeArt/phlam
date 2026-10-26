package build_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	build "github.com/LynnColeArt/phlam/compiler/build"
	"github.com/LynnColeArt/phlam/runtime/bundle"
	abi "github.com/LynnColeArt/phlam/runtime/components"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

var binaries string

func TestMain(m *testing.M) {
	var err error
	binaries, err = os.MkdirTemp("", "phlam-g01-test-tools-")
	if err != nil {
		panic(err)
	}
	for _, name := range []string{"phlam", "phlam-train", "phlam-runtime"} {
		command := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-o", filepath.Join(binaries, name), "../../cmd/"+name)
		if output, err := command.CombinedOutput(); err != nil {
			panic(string(output))
		}
	}
	code := m.Run()
	os.RemoveAll(binaries)
	os.Exit(code)
}
func key() ed25519.PrivateKey {
	seed := sha256.Sum256([]byte("G01 TEST KEY"))
	return ed25519.NewKeyFromSeed(seed[:])
}
func project(t *testing.T, variant string) string {
	t.Helper()
	dir := t.TempDir()
	files, err := os.ReadDir("../../examples/g01/" + variant)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		raw, err := os.ReadFile("../../examples/g01/" + variant + "/" + f.Name())
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, f.Name()), raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}
func opts(root string) build.Options {
	return build.Options{Output: filepath.Join(root, "bundle"), Evidence: filepath.Join(root, "evidence"), Trainer: filepath.Join(binaries, "phlam-train"), Runtime: filepath.Join(binaries, "phlam-runtime"), SigningKey: key(), ImplementationHash: abi.Hash([]byte("G01 test implementation")), Context: context.Background()}
}
func relock(t *testing.T, dir string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "phlam.mod"))
	if err != nil {
		t.Fatal(err)
	}
	var m build.Manifest
	json.Unmarshal(raw, &m)
	lock := build.Lock{Schema: "phlame.lock.v0", Files: map[string]string{}}
	for _, name := range append(m.Sources, m.TrainingPlan) {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		lock.Files[name] = abi.Hash(raw)
	}
	if err := build.Save(filepath.Join(dir, m.Lock), lock); err != nil {
		t.Fatal(err)
	}
}
func mutateJSON(t *testing.T, path string, fn func(map[string]any)) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	json.Unmarshal(raw, &v)
	fn(v)
	if err := build.Save(path, v); err != nil {
		t.Fatal(err)
	}
}
func TestCampaignBuildReproduceAndSourceFree(t *testing.T) {
	deps, err := exec.Command("go", "list", "-deps", "../../cmd/phlam-runtime").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"/compiler/", "/runtime/reference", "/runtime/learned"} {
		if strings.Contains(string(deps), bad) {
			t.Fatal("runtime imports authoring/source machinery", bad)
		}
	}
	symbols, err := exec.Command("go", "tool", "nm", filepath.Join(binaries, "phlam-runtime")).Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(symbols), "components.Train") {
		t.Fatal("optimizer linked into deployed runtime")
	}
	for _, variant := range []string{"original", "renamed", "reordered"} {
		t.Run(variant, func(t *testing.T) {
			dir := project(t, variant)
			first := opts(t.TempDir())
			report, err := build.Build(filepath.Join(dir, "phlam.mod"), first)
			if err != nil {
				t.Fatal(err)
			}
			if report.Status != "built" || len(report.Components) != 3 {
				t.Fatal("incomplete build")
			}
			second := opts(t.TempDir())
			if _, err := build.Build(filepath.Join(dir, "phlam.mod"), second); err != nil {
				t.Fatal(err)
			}
			files, err := os.ReadDir(first.Output)
			if err != nil {
				t.Fatal(err)
			}
			if len(files) != 4 {
				t.Fatal("deployment contains extra files")
			}
			for _, file := range files {
				a, _ := os.ReadFile(filepath.Join(first.Output, file.Name()))
				b, _ := os.ReadFile(filepath.Join(second.Output, file.Name()))
				if !bytes.Equal(a, b) {
					t.Fatal("nonreproducible deployment", file.Name())
				}
			}
			manifest, artifacts, err := bundle.Load(first.Output, key().Public().(ed25519.PublicKey))
			if err != nil {
				t.Fatal(err)
			}
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
			for alias := range report.Components {
				var observed []build.Observation
				raw, _ := os.ReadFile(filepath.Join(first.Evidence, alias, "observations.json"))
				json.Unmarshal(raw, &observed)
				opcount := len(artifacts[alias].Contract.Operations)
				if opcount != 2 {
					t.Fatal("operation metadata missing")
				}
				for _, o := range observed {
					op := artifacts[alias].Contract.Operations[o.Operation]
					want := o.State
					accepted := true
					if op.Arity == 1 {
						want += o.Arg
						switch alias {
						case "accumulator":
							accepted = want >= -16 && want <= 16
						case "saturating":
							if want < -16 {
								want = -16
							}
							if want > 16 {
								want = 16
							}
						}
					}
					if accepted != o.Accepted || accepted && (o.Prediction.Next != want || o.Prediction.Result != want || o.Prediction.Write != (op.Arity == 1)) {
						t.Fatal("independent mathematical fixture disagrees with source", alias, o)
					}
					step := abi.Step{Target: "x", Operation: op.Name}
					if op.Arity == 1 {
						arg := o.Arg
						step.Argument = &abi.Binding{Literal: &arg}
					}
					tx := abi.Transaction{ID: "primitive", Instances: map[string]abi.Instance{"x": {Component: alias, Value: o.State, Version: 1}}, Steps: []abi.Step{step}}
					for name, r := range plans {
						out := r.Run(tx)
						if out.Accepted != accepted {
							t.Fatal(name, "disposition differs", o, out)
						}
						if !accepted {
							if !reflect.DeepEqual(out.States, tx.Instances) || len(out.Results) > 0 || len(out.Writes) > 0 {
								t.Fatal("rejection leaked")
							}
						} else {
							version := uint64(1)
							if op.Arity == 1 {
								version = 2
							}
							if out.States["x"].Value != want || out.States["x"].Version != version || out.Results[0].Value != want {
								t.Fatal("incorrect accepted state/return")
							}
						}
					}
				}
			}
			// Both instances stage writes, including same-value saturation, before failure.
			one := int64(1)
			ref0, ref1 := 0, 1
			tx := abi.Transaction{ID: "late-rollback", Instances: map[string]abi.Instance{"c": {Component: "counter", Value: 0, Version: 1}, "s": {Component: "saturating", Value: 16, Version: 1}}, Steps: []abi.Step{{Target: "c", Operation: "increment", Argument: &abi.Binding{Literal: &one}}, {Target: "s", Operation: "add", Argument: &abi.Binding{From: &ref0}}, {Target: "c", Operation: "increment", Argument: &abi.Binding{From: &ref1}}}}
			for name, r := range plans {
				out := r.Run(tx)
				if out.Accepted || out.FailedStep != 2 || !reflect.DeepEqual(out.States, tx.Instances) || len(out.Writes) > 0 || len(out.Results) > 0 {
					t.Fatal(name, "late rollback failed", out)
				}
				if strings.Contains(name, "certified") && out.Work.Relations != (abi.Work{}) {
					t.Fatal("warm checker evaluated relation")
				}
			}
			for _, kind := range []string{"missing-argument", "forward-result", "outside-domain", "native-refusal", "unknown-target"} {
				bad := tx
				bad.Instances = map[string]abi.Instance{"c": {Component: "counter", Value: 0, Version: 1}, "a": {Component: "accumulator", Value: 15, Version: 1}}
				bad.Steps = []abi.Step{{Target: "c", Operation: "increment", Argument: &abi.Binding{Literal: &one}}}
				switch kind {
				case "missing-argument":
					bad.Steps[0].Argument = nil
				case "forward-result":
					bad.Steps[0].Argument = &abi.Binding{From: &ref1}
				case "outside-domain":
					state := bad.Instances["c"]
					state.Value = 17
					bad.Instances["c"] = state
				case "native-refusal":
					state := bad.Instances["c"]
					state.Value = 3
					bad.Instances["c"] = state
					bad.Steps = append(bad.Steps, abi.Step{Target: "a", Operation: "add", Argument: &abi.Binding{From: &ref0}})
				case "unknown-target":
					bad.Steps[0].Target = "foreign"
				}
				for name, r := range plans {
					out := r.Run(bad)
					if out.Accepted || !reflect.DeepEqual(out.States, bad.Instances) || len(out.Results) > 0 || len(out.Writes) > 0 {
						t.Fatal(name, kind, "negative transaction leaked", out)
					}
				}
			}
			if err := bundle.CheckEntries(manifest, tx); err != nil {
				t.Fatal(err)
			}
			// CLI runner operates with only the deployment files, binary, key and request.
			runtimeDir := t.TempDir()
			for _, file := range files {
				raw, _ := os.ReadFile(filepath.Join(first.Output, file.Name()))
				os.WriteFile(filepath.Join(runtimeDir, file.Name()), raw, 0644)
			}
			raw, _ := os.ReadFile(filepath.Join(binaries, "phlam-runtime"))
			os.WriteFile(filepath.Join(runtimeDir, "runner"), raw, 0755)
			os.WriteFile(filepath.Join(runtimeDir, "key.hex"), []byte(hex.EncodeToString(key().Public().(ed25519.PublicKey))), 0644)
			payload, _ := json.Marshal(map[string]any{"transactions": []abi.Transaction{tx}, "repeats": 1})
			command := exec.Command(filepath.Join(runtimeDir, "runner"), "run", ".", "--key", "key.hex")
			command.Dir = runtimeDir
			command.Stdin = bytes.NewReader(payload)
			out, err := command.CombinedOutput()
			if err != nil {
				t.Fatal(string(out), err)
			}
			var response struct {
				Outcomes []abi.Outcome `json:"outcomes"`
			}
			if err := json.Unmarshal(out, &response); err != nil || len(response.Outcomes) != 1 || response.Outcomes[0].Accepted || response.Outcomes[0].FailedStep != 2 {
				t.Fatal("source-free rollback failed", string(out))
			}
		})
	}
}
func TestInputFailuresAndCandidateFailurePublishNothing(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{"bad-lock", func(t *testing.T, d string) { os.WriteFile(filepath.Join(d, "counter.phlam"), []byte("changed"), 0644) }},
		{"undeclared", func(t *testing.T, d string) {
			mutateJSON(t, filepath.Join(d, "phlam.mod"), func(v map[string]any) { v["components"].([]any)[0].(map[string]any)["source"] = "ambient.phlam" })
		}},
		{"domain", func(t *testing.T, d string) {
			mutateJSON(t, filepath.Join(d, "phlam.mod"), func(v map[string]any) {
				v["components"].([]any)[0].(map[string]any)["domain"].(map[string]any)["state_max"] = 1000
			})
		}},
		{"unknown-recipe", func(t *testing.T, d string) {
			mutateJSON(t, filepath.Join(d, "training.json"), func(v map[string]any) { v["candidates"].([]any)[0].(map[string]any)["kind"] = "application-handler" })
			relock(t, d)
		}},
		{"unsupported-source", func(t *testing.T, d string) {
			p := filepath.Join(d, "counter.phlam")
			raw, _ := os.ReadFile(p)
			os.WriteFile(p, []byte(strings.Replace(string(raw), "$this->value += $delta", "$this->value = eval('1')", 1)), 0644)
			relock(t, d)
		}},
		{"exhausted-search", func(t *testing.T, d string) {
			mutateJSON(t, filepath.Join(d, "training.json"), func(v map[string]any) {
				v["candidates"] = []any{map[string]any{"kind": "relu", "width": 2, "initialization": "legacy", "seeds": []int{404}}}
			})
			relock(t, d)
		}},
		{"timeout", func(t *testing.T, d string) {
			mutateJSON(t, filepath.Join(d, "training.json"), func(v map[string]any) { v["candidate_timeout_ms"] = 1 })
			relock(t, d)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := project(t, "original")
			tc.mutate(t, d)
			options := opts(t.TempDir())
			r, err := build.Build(filepath.Join(d, "phlam.mod"), options)
			if err == nil || r.Status != "failed" {
				t.Fatal("invalid build succeeded", r, err)
			}
			if _, e := os.Stat(options.Output); !os.IsNotExist(e) {
				t.Fatal("failed build published output")
			}
			raw, e := os.ReadFile(filepath.Join(options.Evidence, "report.json"))
			if e != nil || !strings.Contains(string(raw), `"status": "failed"`) {
				t.Fatal("failed report missing")
			}
			if tc.name == "unsupported-source" && !strings.Contains(err.Error(), "B_UNSUPPORTED") {
				t.Fatal("source diagnostic missing", err)
			}
		})
	}
	d := project(t, "original")
	o := opts(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	o.Context = ctx
	if _, err := build.Build(filepath.Join(d, "phlam.mod"), o); err == nil {
		t.Fatal("cancelled build succeeded")
	}
	if _, err := os.Stat(o.Output); !os.IsNotExist(err) {
		t.Fatal("cancelled build published")
	}
	// A fixed wall budget does not rely on training cooperatively polling context.
}

func TestMidTrainingCancellationDoesNotActivate(t *testing.T) {
	d := project(t, "original")
	root := t.TempDir()
	options := opts(root)
	marker := filepath.Join(root, "started")
	trainer := filepath.Join(root, "blocking-trainer")
	if err := os.WriteFile(trainer, []byte("#!/bin/sh\n: > "+marker+"\nexec /bin/sleep 20\n"), 0755); err != nil {
		t.Fatal(err)
	}
	options.Trainer = trainer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	options.Context = ctx
	done := make(chan error, 1)
	go func() { _, err := build.Build(filepath.Join(d, "phlam.mod"), options); done <- err }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("trainer never started")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled active build succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("training process was not stopped")
	}
	if _, err := os.Stat(options.Output); !os.IsNotExist(err) {
		t.Fatal("interrupted build published a deployment")
	}
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".phlam-build-") {
			t.Fatal("staging directory leaked after handled cancellation")
		}
	}
}
