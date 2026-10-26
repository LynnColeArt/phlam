package bundle

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	abi "github.com/LynnColeArt/phlam/runtime/components"
	"os"
	"path/filepath"
	"testing"
)

func TestIntegrityBindingsAndFreshCertification(t *testing.T) {
	seed := sha256.Sum256([]byte("bundle test key"))
	key := ed25519.NewKeyFromSeed(seed[:])
	public := key.Public().(ed25519.PublicKey)
	oldKey, err := abi.PublicKey("../../experiments/components/run-0001/trusted-key.hex")
	if err != nil {
		t.Fatal(err)
	}
	original, _, err := abi.Load("../../experiments/components/run-0001/counter.json", oldKey)
	if err != nil {
		t.Fatal(err)
	}
	fixture := func(t *testing.T, a abi.Artifact, m Manifest) string {
		t.Helper()
		dir := t.TempDir()
		if err := abi.Freeze(filepath.Join(dir, "model.json"), a, key); err != nil {
			t.Fatal(err)
		}
		if err := SignManifest(filepath.Join(dir, "bundle.json"), m, key); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	m := Manifest{Schema: Schema, ABI: abi.ABI, ArgumentPolicy: ArgumentPolicy, Module: Identity{Name: "tests/counter", Version: "1"}, Modules: map[string]Binding{"a": {File: "model.json", Root: abi.Root(original), Entries: []string{"read"}}}}
	dir := fixture(t, original, m)
	loaded, artifacts, err := Load(dir, public)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := abi.NewRegistry(artifacts, "learned", "certified"); err != nil {
		t.Fatal(err)
	}
	tx := abi.Transaction{Instances: map[string]abi.Instance{"x": {Component: "a", Value: 0, Version: 1}}, Steps: []abi.Step{{Target: "x", Operation: "read"}}}
	if err := CheckEntries(loaded, tx); err != nil {
		t.Fatal(err)
	}
	tx.Steps[0].Operation = "increment"
	if err := CheckEntries(loaded, tx); err == nil {
		t.Fatal("unexported entry accepted")
	}
	if _, _, err := Load(dir, oldKey); err == nil {
		t.Fatal("wrong trust key accepted")
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "bundle.json"))
	var document map[string]any
	json.Unmarshal(raw, &document)
	document["payload"].(map[string]any)["module"].(map[string]any)["name"] = "swapped"
	raw, _ = json.Marshal(document)
	if err := os.Chmod(filepath.Join(dir, "bundle.json"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bundle.json"), raw, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(dir, public); err == nil {
		t.Fatal("unsigned alias/identity edit accepted")
	}
	m.ABI = "other"
	dir = fixture(t, original, m)
	if _, _, err := Load(dir, public); err == nil {
		t.Fatal("wrong ABI accepted")
	}
	m.ABI = abi.ABI
	m.Modules["a"] = Binding{File: "model.json", Root: abi.Hash([]byte("wrong root")), Entries: []string{"read"}}
	dir = fixture(t, original, m)
	if _, _, err := Load(dir, public); err == nil {
		t.Fatal("wrong pinned root accepted")
	}
	bad := original
	bad.Model.Weights = append([][]float64(nil), original.Model.Weights...)
	bad.Model.Weights[0] = append([]float64(nil), bad.Model.Weights[0]...)
	bad.Model.Weights[0][0] += 1
	m.Modules["a"] = Binding{File: "model.json", Root: abi.Root(bad), Entries: []string{"read"}}
	dir = fixture(t, bad, m)
	_, artifacts, err = Load(dir, public)
	if err != nil {
		t.Fatal(err)
	}
	if r, _, err := abi.NewRegistry(artifacts, "learned", "certified"); err == nil || r != nil {
		t.Fatal("valid signature substituted for semantics")
	}
	dir = fixture(t, original, Manifest{Schema: Schema, ABI: abi.ABI, ArgumentPolicy: ArgumentPolicy, Module: m.Module, Modules: map[string]Binding{"a": {File: "model.json", Root: abi.Root(original), Entries: []string{"read"}}}})
	raw, _ = os.ReadFile(filepath.Join(dir, "model.json"))
	raw[len(raw)/2] ^= 1
	if err := os.Chmod(filepath.Join(dir, "model.json"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "model.json"), raw, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(dir, public); err == nil {
		t.Fatal("tampered model accepted")
	}
}
