package learned

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

	"github.com/LynnColeArt/phlam/compiler/contracts"
)

type Artifact struct {
	Schema             string           `json:"schema"`
	Architecture       string           `json:"architecture"`
	PrimitiveABI       string           `json:"primitive_abi"`
	Domain             string           `json:"domain"`
	Seed               int64            `json:"seed"`
	Epochs             int              `json:"epochs"`
	LearningRate       float64          `json:"learning_rate"`
	SourceHash         string           `json:"source_sha256"`
	TrainingHash       string           `json:"training_sha256"`
	SplitsHash         string           `json:"splits_sha256"`
	ImplementationHash string           `json:"implementation_sha256"`
	Toolchain          string           `json:"toolchain"`
	SigningPolicy      string           `json:"signing_policy"`
	Model              Model            `json:"model"`
	Contracts          contracts.Bundle `json:"contracts"`
}
type Envelope struct {
	Payload   json.RawMessage `json:"payload"`
	SHA256    string          `json:"sha256"`
	Signature string          `json:"signature_ed25519"`
}

func Hash(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }
func StrictJSON(data []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("expected one JSON document")
	}
	return nil
}
func Freeze(path string, a Artifact, key ed25519.PrivateKey) error {
	if len(key) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid signing key")
	}
	payload, err := json.Marshal(a)
	if err != nil {
		return err
	}
	e := Envelope{Payload: payload, SHA256: Hash(payload), Signature: hex.EncodeToString(ed25519.Sign(key, payload))}
	data, err := json.Marshal(e)
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
func Load(path string, trusted ed25519.PublicKey) (*Executor, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 256*1024+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 256*1024 {
		return nil, fmt.Errorf("artifact size limit exceeded")
	}
	var e Envelope
	if err := StrictJSON(data, &e); err != nil {
		return nil, err
	}
	sig, err := hex.DecodeString(e.Signature)
	if err != nil {
		return nil, err
	}
	if len(trusted) != ed25519.PublicKeySize || Hash(e.Payload) != e.SHA256 || !ed25519.Verify(trusted, e.Payload, sig) {
		return nil, fmt.Errorf("artifact integrity/signature check failed")
	}
	var a Artifact
	if err := StrictJSON(e.Payload, &a); err != nil {
		return nil, err
	}
	if a.Schema != "phlam.counter.learned.v0" || a.Architecture != "linear-4x3-round-v0" || a.PrimitiveABI != "phlam.int64.v0" || a.Domain != "state[-64,64];delta[-16,16]" {
		return nil, fmt.Errorf("unsupported artifact schema, architecture, ABI or domain")
	}
	for _, row := range a.Model.Weights {
		for _, w := range row {
			if math.IsNaN(w) || math.IsInf(w, 0) || math.Abs(w) > 100 {
				return nil, fmt.Errorf("invalid weight")
			}
		}
	}
	if a.Contracts.Schema != "phlam.contracts.v0" || a.Contracts.SourceHash != a.SourceHash || len(a.Contracts.Methods) != 2 {
		return nil, fmt.Errorf("invalid contract bundle")
	}
	seen := map[string]bool{}
	for _, c := range a.Contracts.Methods {
		if c.ID != "Counter.read" && c.ID != "Counter.increment" || seen[c.ID] || c.SourceHash != a.SourceHash || c.Status != "inferred" || c.Domain != "int64_state_and_arguments" {
			return nil, fmt.Errorf("invalid operation binding")
		}
		seen[c.ID] = true
		hash := c.Hash
		c.Hash = ""
		raw, _ := json.Marshal(c)
		if Hash(raw) != hash {
			return nil, fmt.Errorf("contract hash mismatch")
		}
		if len(c.Fields) != 1 || c.Fields[0] != "value" || len(c.Parameters) > 1 || len(c.Checks) > 32 || len(c.Updates) > 32 || len(c.Post) > 1 {
			return nil, fmt.Errorf("contract size or frame invalid")
		}
		if c.ID == "Counter.increment" && (len(c.Parameters) != 1 || c.Parameters[0].Name != "delta") || c.ID == "Counter.read" && len(c.Parameters) != 0 {
			return nil, fmt.Errorf("parameter binding mismatch")
		}
		count := 0
		var term func(*contracts.Term, int) error
		term = func(t *contracts.Term, depth int) error {
			if t == nil {
				return nil
			}
			count++
			if depth > 32 || count > 512 {
				return fmt.Errorf("contract term budget exceeded")
			}
			switch t.Op {
			case "int", "null", "add", "sub", "mul", "neg":
			case "slot":
				if t.Name != "value" {
					return fmt.Errorf("unbound slot")
				}
			case "arg":
				if t.Name != "delta" {
					return fmt.Errorf("unbound argument")
				}
			default:
				return fmt.Errorf("unbound primitive")
			}
			if err := term(t.Left, depth+1); err != nil {
				return err
			}
			return term(t.Right, depth+1)
		}
		if err := term(c.Return, 0); err != nil {
			return nil, err
		}
		for name, t := range c.Post {
			if name != "value" {
				return nil, fmt.Errorf("unbound post slot")
			}
			if err := term(t, 0); err != nil {
				return nil, err
			}
		}
		for _, t := range c.Checks {
			if err := term(t, 0); err != nil {
				return nil, err
			}
		}
		for _, u := range c.Updates {
			if u.Slot != "value" {
				return nil, fmt.Errorf("unbound write")
			}
			if err := term(u.Value, 0); err != nil {
				return nil, err
			}
		}
	}
	for _, h := range []string{a.SourceHash, a.TrainingHash, a.SplitsHash, a.ImplementationHash} {
		decoded, err := hex.DecodeString(h)
		if err != nil || len(decoded) != 32 {
			return nil, fmt.Errorf("invalid lineage digest")
		}
	}
	return &Executor{artifact: a}, nil
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

// Fixed-array JSON decoding otherwise silently ignores excess elements.
func (m *Model) UnmarshalJSON(data []byte) error {
	var shape struct {
		Weights [][]float64 `json:"weights"`
	}
	if err := StrictJSON(data, &shape); err != nil {
		return err
	}
	if len(shape.Weights) != 3 {
		return fmt.Errorf("model requires three output rows")
	}
	for h, row := range shape.Weights {
		if len(row) != 4 {
			return fmt.Errorf("model requires four input features")
		}
		for j, w := range row {
			m.Weights[h][j] = w
		}
	}
	return nil
}
