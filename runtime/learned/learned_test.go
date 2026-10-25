package learned

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LynnColeArt/phlam/compiler/contracts"
	"github.com/LynnColeArt/phlam/runtime/reference"
	"github.com/LynnColeArt/phlam/semantics"
)

func fixture(t *testing.T) (Artifact, ed25519.PrivateKey) {
	t.Helper()
	source, err := os.ReadFile("../../examples/counter/counter.phlam")
	if err != nil {
		t.Fatal(err)
	}
	labels := []Label{}
	for state := int64(-32); state <= 32; state += 4 {
		for _, op := range []string{"Counter.read", "Counter.increment"} {
			lo, hi := int64(-8), int64(8)
			if op == "Counter.read" {
				lo, hi = 0, 0
			}
			for delta := lo; delta <= hi; delta += 2 {
				r, err := reference.New(string(source), reference.Options{})
				if err != nil {
					t.Fatal(err)
				}
				h, err := r.Construct("Counter", semantics.Integer(state))
				if err != nil {
					t.Fatal(err)
				}
				method := "read"
				args := []semantics.Value{}
				if op == "Counter.increment" {
					method = "increment"
					args = append(args, semantics.Integer(delta))
				}
				result, err := r.Call(h, method, args...)
				if err != nil {
					t.Fatal(err)
				}
				labels = append(labels, Label{Case: Case{Operation: op, State: state, Delta: delta}, Next: r.Objects()[0].Fields["value"].Int, Result: result.Int, Write: op == "Counter.increment"})
			}
		}
	}
	model, initial, final, err := Train(labels, 20261025)
	if err != nil {
		t.Fatal(err)
	}
	again, _, _, err := Train(labels, 20261025)
	if err != nil || again != model || final >= initial || final > 1e-12 {
		t.Fatal("training did not converge reproducibly", initial, final, err)
	}
	r, err := reference.New(string(source), reference.Options{})
	if err != nil {
		t.Fatal(err)
	}
	bundle := r.Contracts()
	kept := []contracts.Contract{}
	for _, c := range bundle.Methods {
		if c.ID != "Counter.init" {
			kept = append(kept, c)
		}
	}
	bundle.Methods = kept
	a := Artifact{Schema: "phlam.counter.learned.v0", Architecture: "linear-4x3-round-v0", PrimitiveABI: "phlam.int64.v0", Domain: "state[-64,64];delta[-16,16]", SourceHash: r.SourceHash(), TrainingHash: Hash([]byte("test training")), SplitsHash: Hash([]byte("test split")), ImplementationHash: Hash([]byte("test implementation")), Model: model, Contracts: bundle}
	seed := sha256.Sum256([]byte("unit test signing key"))
	return a, ed25519.NewKeyFromSeed(seed[:])
}
func TestFrozenReloadedModelAndChecker(t *testing.T) {
	a, key := fixture(t)
	path := filepath.Join(t.TempDir(), "artifact.json")
	if err := Freeze(path, a, key); err != nil {
		t.Fatal(err)
	}
	if Freeze(path, a, key) == nil {
		t.Fatal("overwrote frozen artifact")
	}
	e, err := Load(path, key.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []Case{{Operation: "Counter.read", State: 63}, {Operation: "Counter.increment", State: 63, Delta: 15}, {Operation: "Counter.increment", State: -64, Delta: -16}, {Operation: "Counter.increment", State: 7, Delta: 0}} {
		got := e.Run(c, "learned")
		want := c.State
		write := c.Operation == "Counter.increment"
		if write {
			want += c.Delta
		}
		if !got.Accepted || got.State != want || got.Result != want || got.Write != write {
			t.Fatal("wrong learned result", c, got)
		}
		operations := uint64(0)
		if write {
			operations = 4
		}
		if got.CheckerWork.IntegerPrimitives != operations || got.Timing.CheckerCalls != 1 || got.Timing.CheckerNS <= 0 {
			t.Fatal("exact checker work not measured", got)
		}
		zero := e.Run(c, "zero")
		if c.State != 0 && zero.Accepted {
			t.Fatal("zero model bypassed checker")
		}
		if !zero.Accepted && (zero.State != c.State || zero.Version != 1) {
			t.Fatal("rejected proposal mutated state")
		}
	}
	for _, c := range []Case{{Operation: "Counter.increment", State: 65, Delta: 1}, {Operation: "Counter.increment", State: 1, Delta: 17}, {Operation: "Counter.read", State: 1, Delta: 1}, {Operation: "Other.run"}} {
		got := e.Run(c, "learned")
		if got.Accepted || got.Timing.CheckerCalls != 0 || got.Version != 1 || got.State != c.State {
			t.Fatal("domain gate failed", got)
		}
	}
	// A failed proposal does not poison the executor or change its frozen weights.
	if !e.Run(Case{Operation: "Counter.increment", State: 63, Delta: 15}, "learned").Accepted {
		t.Fatal("model mutated during ablation")
	}
}
func TestArtifactIntegritySchemaAndShape(t *testing.T) {
	a, key := fixture(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact.json")
	if err := Freeze(path, a, key); err != nil {
		t.Fatal(err)
	}
	other := sha256.Sum256([]byte("other key"))
	wrong := ed25519.NewKeyFromSeed(other[:])
	if _, err := Load(path, wrong.Public().(ed25519.PublicKey)); err == nil {
		t.Fatal("untrusted signature accepted")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var envelope Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.Payload = []byte(strings.Replace(string(envelope.Payload), "linear-4x3-round-v0", "linear-4x3-round-v9", 1))
	tampered, _ := json.Marshal(envelope)
	bad := filepath.Join(dir, "tampered.json")
	if err := os.WriteFile(bad, tampered, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(bad, key.Public().(ed25519.PublicKey)); err == nil {
		t.Fatal("tampered payload accepted")
	}
	for i, change := range []func(*Artifact){
		func(a *Artifact) { a.PrimitiveABI = "unknown" },
		func(a *Artifact) { a.Contracts.Methods[0].Hash = "bad" },
		func(a *Artifact) { a.Model.Weights[0][0] = 101 },
	} {
		copy := a
		raw, _ := json.Marshal(a)
		if err := json.Unmarshal(raw, &copy); err != nil {
			t.Fatal(err)
		}
		change(&copy)
		bad := filepath.Join(dir, string(rune('a'+i))+".json")
		if err := Freeze(bad, copy, key); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(bad, key.Public().(ed25519.PublicKey)); err == nil {
			t.Fatal("bad signed artifact accepted")
		}
	}
	var m Model
	for _, raw := range []string{`{"weights":[[1,2,3,4,5],[1,2,3,4],[1,2,3,4]]}`, `{"weights":[[1,2,3,4]]}`, `{"weights":[[1,2,3,4],[1,2,3,4],[1,2,3,4]],"extra":0}`} {
		if json.Unmarshal([]byte(raw), &m) == nil {
			t.Fatal("invalid model shape accepted")
		}
	}
	oversized := filepath.Join(dir, "huge.json")
	if err := os.WriteFile(oversized, make([]byte, 256*1024+1), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(oversized, key.Public().(ed25519.PublicKey)); err == nil {
		t.Fatal("oversized artifact accepted")
	}
}
