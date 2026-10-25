package components

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func constant(i int64) *Expr { return &Expr{Op: "int", Int: i} }
func syntheticArtifact() Artifact {
	truth := &Expr{Op: "bool", Bool: true}
	slot := &Expr{Op: "slot"}
	sum := &Expr{Op: "add", Left: slot, Right: &Expr{Op: "arg"}}
	return Artifact{Schema: "phlame.scalar-module.v0", ABI: ABI, Contract: Contract{Class: "Synthetic", Slot: "owned", SourceHash: Hash([]byte("source")), Operations: []Operation{
		{Name: "bump", Arity: 1, Paths: []Path{{Guard: truth, Post: sum, Writes: []*Expr{sum}, Checks: []*Expr{sum}, Return: sum}}},
		{Name: "peek", Arity: 0, Paths: []Path{{Guard: truth, Post: slot, Return: slot}}},
	}}, Domain: Domain{StateMin: -2, StateMax: 2, ArgMin: -1, ArgMax: 1}, Model: Model{Weights: [][]float64{{0, 1, 1, 0, 0}, {0, 1, 1, 0, 0}, {0, 0, 0, 1, 0}}}, TrainingHash: Hash([]byte("train")), VerificationHash: Hash([]byte("observed")), ImplementationHash: Hash([]byte("implementation"))}
}
func lit(n int64) *Binding { return &Binding{Literal: &n} }
func ref(n int) *Binding   { return &Binding{From: &n} }
func TestGenericRegistryValuePassingAndFrames(t *testing.T) {
	a := syntheticArtifact()
	b := syntheticArtifact()
	b.Contract.Class = "Other"
	b.Contract.Slot = "separate"
	artifacts := map[string]Artifact{"alpha": a, "beta": b}
	registry, proofs, err := NewRegistry(artifacts, "learned", "certified")
	if err != nil {
		t.Fatal(err)
	}
	if !proofs["alpha"].Certified || !proofs["beta"].Certified || proofs["alpha"].Key == proofs["beta"].Key {
		t.Fatal("certificates not component-bound")
	}
	tx := Transaction{Instances: map[string]Instance{"first": {Component: "alpha", Value: 0, Version: 1}, "second": {Component: "beta", Value: 0, Version: 1}}, Steps: []Step{{Target: "first", Operation: "bump", Argument: lit(1)}, {Target: "second", Operation: "bump", Argument: ref(0)}, {Target: "first", Operation: "peek"}}}
	out := registry.Run(tx)
	if !out.Accepted || out.States["first"].Value != 1 || out.States["second"].Value != 1 || out.States["first"].Version != 2 || out.States["second"].Version != 2 || out.Results[1].Producer != "second" || out.Results[1].Type != "int64" || out.Work.Relations.Arithmetic != 0 {
		t.Fatal("generic value passing failed", out)
	}
	if out.Writes[0].Slot != "owned" || out.Writes[1].Slot != "separate" {
		t.Fatal("component frame mismatch")
	}
	// Mutating caller-owned artifacts after registration cannot change the sealed
	// model, source relation, admission rows or any certificate.
	artifacts["alpha"].Model.Weights[0][0] = 1
	artifacts["beta"].Contract.Operations[0].Paths[0].Return.Op = "bool"
	if !registry.Run(tx).Accepted {
		t.Fatal("registry retains caller-owned mutable data")
	}
	// A one-component operation cannot touch a peer even when both are in scope.
	tx.Steps = tx.Steps[:1]
	out = registry.Run(tx)
	if !out.Accepted || out.States["second"] != tx.Instances["second"] {
		t.Fatal("peer frame changed", out)
	}
}
func TestRollbackBindingsAndReplacement(t *testing.T) {
	a := syntheticArtifact()
	registry, proofs, err := NewRegistry(map[string]Artifact{"a": a, "b": a}, "learned", "certified")
	if err != nil {
		t.Fatal(err)
	}
	base := Transaction{Instances: map[string]Instance{"x": {Component: "a", Value: 0, Version: 1}, "y": {Component: "b", Value: 2, Version: 1}}, Steps: []Step{{Target: "x", Operation: "bump", Argument: lit(1)}, {Target: "y", Operation: "bump", Argument: lit(1)}, {Target: "x", Operation: "bump", Argument: ref(1)}}}
	out := registry.Run(base)
	if out.Accepted || !reflect.DeepEqual(out.States, base.Instances) || len(out.Results) != 0 || len(out.Writes) != 0 {
		t.Fatal("rollback leaked cross-component writes", out)
	}
	cases := []Step{{Target: "missing", Operation: "peek"}, {Target: "x", Operation: "bump", Argument: ref(4)}, {Target: "x", Operation: "bump", Argument: &Binding{Literal: lit(1).Literal, From: ref(0).From}}, {Target: "x", Operation: "bump"}, {Target: "x", Operation: "peek", Argument: lit(0)}, {Target: "x", Operation: "unknown"}}
	for _, last := range cases {
		tx := base
		tx.Steps = []Step{base.Steps[0], last}
		out := registry.Run(tx)
		if out.Accepted || !reflect.DeepEqual(out.States, tx.Instances) {
			t.Fatal("invalid binding/frame committed", last, out)
		}
	}
	bad := syntheticArtifact()
	bad.Model.Weights[0][0] = 1
	_, newProofs, err := NewRegistry(map[string]Artifact{"a": a, "b": bad}, "learned", "certified")
	if err == nil || !newProofs["a"].Certified || newProofs["a"].Key != proofs["a"].Key || newProofs["b"].Key == proofs["b"].Key || newProofs["b"].Certified {
		t.Fatal("replacement reused another component certificate", newProofs, err)
	}
	for _, engine := range []string{"zero", "shuffled"} {
		if _, _, err := NewRegistry(map[string]Artifact{"a": a}, engine, "certified"); err == nil {
			t.Fatal("invalid model certified")
		}
	}
	if new(Registry).Run(base).Accepted {
		t.Fatal("uninitialized registry committed")
	}
}
func TestSignedArtifactTamperShapeAndBinding(t *testing.T) {
	a := syntheticArtifact()
	seed := sha256.Sum256([]byte("public test key"))
	key := ed25519.NewKeyFromSeed(seed[:])
	dir := t.TempDir()
	path := filepath.Join(dir, "a.json")
	if err := Freeze(path, a, key); err != nil {
		t.Fatal(err)
	}
	loaded, root, err := Load(path, key.Public().(ed25519.PublicKey))
	if err != nil || root != Root(a) || !reflect.DeepEqual(loaded, a) {
		t.Fatal("artifact reload mismatch", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var envelope Envelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.Payload = bytes.Replace(envelope.Payload, []byte("Synthetic"), []byte("Changed"), 1)
	tampered, _ := json.Marshal(envelope)
	bad := filepath.Join(dir, "tamper.json")
	if err := os.WriteFile(bad, tampered, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(bad, key.Public().(ed25519.PublicKey)); err == nil {
		t.Fatal("corrupt module accepted")
	}
	a.Model.Weights[0] = append(a.Model.Weights[0], 0)
	if ValidateArtifact(a) == nil {
		t.Fatal("extra tensor dimension accepted")
	}
}
