package components

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

type Artifact struct {
	Initialization     string   `json:"training_initialization,omitempty"`
	PrefitEpochs       int      `json:"prefit_epochs,omitempty"`
	VerificationHash   string   `json:"oracle_domain_sha256"`
	Schema             string   `json:"schema"`
	ABI                string   `json:"abi"`
	Contract           Contract `json:"contract"`
	Domain             Domain   `json:"domain"`
	Model              Model    `json:"model"`
	Seed               int64    `json:"seed"`
	Epochs             int      `json:"epochs"`
	TrainingHash       string   `json:"training_sha256"`
	ImplementationHash string   `json:"implementation_sha256"`
	Toolchain          string   `json:"toolchain"`
}
type Envelope struct {
	Payload   json.RawMessage `json:"payload"`
	Hash      string          `json:"sha256"`
	Signature string          `json:"signature_ed25519"`
}

func Hash(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func StrictJSON(data []byte, v any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return fmt.Errorf("expected one JSON document")
	}
	return nil
}
func Freeze(path string, a Artifact, key ed25519.PrivateKey) error {
	if len(key) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid key")
	}
	if err := ValidateArtifact(a); err != nil {
		return err
	}
	payload, err := json.Marshal(a)
	if err != nil {
		return err
	}
	data, err := json.Marshal(Envelope{Payload: payload, Hash: Hash(payload), Signature: hex.EncodeToString(ed25519.Sign(key, payload))})
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0444)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func Load(path string, key ed25519.PublicKey) (Artifact, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return Artifact{}, "", err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 256*1024+1))
	if err != nil {
		return Artifact{}, "", err
	}
	if len(data) > 256*1024 {
		return Artifact{}, "", fmt.Errorf("artifact size limit")
	}
	var envelope Envelope
	if err := StrictJSON(data, &envelope); err != nil {
		return Artifact{}, "", err
	}
	signature, err := hex.DecodeString(envelope.Signature)
	if err != nil || len(key) != ed25519.PublicKeySize || Hash(envelope.Payload) != envelope.Hash || !ed25519.Verify(key, envelope.Payload, signature) {
		return Artifact{}, "", fmt.Errorf("artifact integrity failure")
	}
	var a Artifact
	if err := StrictJSON(envelope.Payload, &a); err != nil {
		return a, "", err
	}
	if err := ValidateArtifact(a); err != nil {
		return a, "", err
	}
	return a, envelope.Hash, nil
}
func PublicKey(path string) (ed25519.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	key, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil || len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key")
	}
	return ed25519.PublicKey(key), nil
}
func ValidateArtifact(a Artifact) error {
	if a.Schema != "phlame.scalar-module.v0" || a.ABI != ABI {
		return fmt.Errorf("unknown module schema or ABI")
	}
	switch a.Initialization {
	case "", "legacy", "paired-centered":
		if a.PrefitEpochs != 0 {
			return fmt.Errorf("unexpected prefit epoch count")
		}
	case "paired-affine":
		if a.PrefitEpochs != Epochs {
			return fmt.Errorf("invalid affine prefit lineage")
		}
	default:
		return fmt.Errorf("unknown training initialization")
	}
	d := a.Domain
	if d.StateMin < -64 || d.StateMax > 64 || d.StateMin > d.StateMax || d.ArgMin < -16 || d.ArgMax > 16 || d.ArgMin > d.ArgMax {
		return fmt.Errorf("invalid finite domain")
	}
	c := a.Contract
	if c.Class == "" || c.Slot == "" || len(c.Operations) == 0 || len(c.Operations) > 8 {
		return fmt.Errorf("invalid component frame")
	}
	for _, digest := range []string{c.SourceHash, a.TrainingHash, a.VerificationHash, a.ImplementationHash} {
		b, err := hex.DecodeString(digest)
		if err != nil || len(b) != 32 {
			return fmt.Errorf("invalid lineage digest")
		}
	}
	if len(a.Model.Weights) != 3 {
		return fmt.Errorf("model output shape mismatch")
	}
	for _, row := range a.Model.Weights {
		if len(row) != 3+len(c.Operations)+len(a.Model.Hidden) {
			return fmt.Errorf("model feature shape mismatch")
		}
		for _, w := range row {
			if math.IsNaN(w) || math.IsInf(w, 0) || math.Abs(w) > 100 {
				return fmt.Errorf("invalid model weight")
			}
		}
	}
	if len(a.Model.Hidden) > 32 {
		return fmt.Errorf("hidden layer budget")
	}
	for _, row := range a.Model.Hidden {
		if len(row) != 3+len(c.Operations) {
			return fmt.Errorf("hidden feature shape mismatch")
		}
		for _, w := range row {
			if math.IsNaN(w) || math.IsInf(w, 0) || math.Abs(w) > 100 {
				return fmt.Errorf("invalid hidden weight")
			}
		}
	}
	seen := map[string]bool{}
	nodes := 0
	var expr func(*Expr, int) error
	expr = func(e *Expr, depth int) error {
		if e == nil {
			return fmt.Errorf("missing expression")
		}
		nodes++
		if depth > 32 || nodes > 2048 {
			return fmt.Errorf("expression budget exceeded")
		}
		switch e.Op {
		case "slot", "arg", "int", "bool":
			if e.Left != nil || e.Right != nil {
				return fmt.Errorf("invalid leaf")
			}
			return nil
		case "not":
			if e.Right != nil {
				return fmt.Errorf("invalid unary expression")
			}
			return expr(e.Left, depth+1)
		case "add", "sub", "mul", "lt", "gt", "le", "ge", "eq", "ne", "and", "or":
			if err := expr(e.Left, depth+1); err != nil {
				return err
			}
			return expr(e.Right, depth+1)
		default:
			return fmt.Errorf("unbound expression primitive")
		}
	}
	for _, op := range c.Operations {
		if op.Name == "" || seen[op.Name] || op.Arity < 0 || op.Arity > 1 || len(op.Paths) == 0 || len(op.Paths) > 16 {
			return fmt.Errorf("invalid operation")
		}
		seen[op.Name] = true
		for _, p := range op.Paths {
			if len(p.Writes) > 1 || len(p.Checks) > 64 {
				return fmt.Errorf("effect or check budget")
			}
			for _, e := range []*Expr{p.Guard, p.Post, p.Return} {
				if err := expr(e, 0); err != nil {
					return err
				}
			}
			for _, e := range append(append([]*Expr(nil), p.Checks...), p.Writes...) {
				if err := expr(e, 0); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
