// Package bundle loads signed scalar deployment bundles without compiler/oracle code.
package bundle

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	abi "github.com/LynnColeArt/phlam/runtime/components"
	"io"
	"os"
	"path/filepath"
	"sort"
)

const Schema = "phlame.bundle.v0"
const ArgumentPolicy = "explicit-int64-v0"

type Identity struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
type Binding struct {
	File    string   `json:"file"`
	Root    string   `json:"module_sha256"`
	Entries []string `json:"entries"`
}
type Manifest struct {
	Schema         string             `json:"schema"`
	ABI            string             `json:"abi"`
	ArgumentPolicy string             `json:"argument_policy"`
	Module         Identity           `json:"module"`
	Modules        map[string]Binding `json:"modules"`
}

func SignManifest(path string, m Manifest, key ed25519.PrivateKey) error {
	if len(key) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid signing key")
	}
	payload, err := json.Marshal(m)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(abi.Envelope{Payload: payload, Hash: abi.Hash(payload), Signature: hex.EncodeToString(ed25519.Sign(key, payload))})
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0444)
	if err != nil {
		return err
	}
	_, err = f.Write(raw)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func ReadManifest(dir string, key ed25519.PublicKey) (Manifest, error) {
	var m Manifest
	file, err := os.Open(filepath.Join(dir, "bundle.json"))
	if err != nil {
		return m, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, 32769))
	if err != nil {
		return m, err
	}
	if len(raw) > 32768 {
		return m, fmt.Errorf("bundle manifest limit")
	}
	var envelope abi.Envelope
	if err := abi.StrictJSON(raw, &envelope); err != nil {
		return m, err
	}
	signature, err := hex.DecodeString(envelope.Signature)
	if err != nil || len(key) != ed25519.PublicKeySize || abi.Hash(envelope.Payload) != envelope.Hash || !ed25519.Verify(key, envelope.Payload, signature) {
		return m, fmt.Errorf("bundle signature/integrity failure")
	}
	if err := abi.StrictJSON(envelope.Payload, &m); err != nil {
		return m, err
	}
	if m.Schema != Schema || m.ABI != abi.ABI || m.ArgumentPolicy != ArgumentPolicy || m.Module.Name == "" || m.Module.Version == "" || len(m.Modules) < 1 || len(m.Modules) > 8 {
		return m, fmt.Errorf("unsupported bundle/schema/ABI")
	}
	return m, nil
}
func Load(dir string, key ed25519.PublicKey) (Manifest, map[string]abi.Artifact, error) {
	m, err := ReadManifest(dir, key)
	if err != nil {
		return m, nil, err
	}
	artifacts := map[string]abi.Artifact{}
	for alias, b := range m.Modules {
		if alias == "" || b.File == "" || filepath.Base(b.File) != b.File || b.File == "." || b.File == ".." || len(b.Entries) == 0 {
			return m, nil, fmt.Errorf("invalid bundle binding")
		}
		info, err := os.Lstat(filepath.Join(dir, b.File))
		if err != nil {
			return m, nil, err
		}
		if !info.Mode().IsRegular() {
			return m, nil, fmt.Errorf("module must be a regular file")
		}
		a, root, err := abi.Load(filepath.Join(dir, b.File), key)
		if err != nil {
			return m, nil, err
		}
		if root != b.Root {
			return m, nil, fmt.Errorf("module root mismatch for %s", alias)
		}
		ops := map[string]bool{}
		for _, op := range a.Contract.Operations {
			ops[op.Name] = true
		}
		seen := map[string]bool{}
		for _, entry := range b.Entries {
			if !ops[entry] || seen[entry] {
				return m, nil, fmt.Errorf("invalid/duplicate entry %s", entry)
			}
			seen[entry] = true
		}
		artifacts[alias] = a
	}
	return m, artifacts, nil
}

// CheckEntries narrows invocation to signed bundle entries; all scalar runtime
// frame, binding, version, domain and transactional checks still run afterward.
type EntryWork struct {
	Instances   uint64 `json:"instance_bindings"`
	Targets     uint64 `json:"step_targets"`
	Comparisons uint64 `json:"entry_comparisons"`
}

func CheckEntries(m Manifest, tx abi.Transaction) error {
	_, err := CheckEntriesMeasured(m, tx)
	return err
}
func CheckEntriesMeasured(m Manifest, tx abi.Transaction) (work EntryWork, err error) {
	if len(tx.Instances) < 1 || len(tx.Instances) > 8 || len(tx.Steps) < 1 || len(tx.Steps) > 16 {
		return work, fmt.Errorf("invalid transaction bounds")
	}
	for _, state := range tx.Instances {
		work.Instances++
		if _, ok := m.Modules[state.Component]; !ok {
			return work, fmt.Errorf("unbound component")
		}
	}
	for _, step := range tx.Steps {
		work.Targets++
		state, ok := tx.Instances[step.Target]
		if !ok {
			return work, fmt.Errorf("unknown instance")
		}
		found := false
		for _, name := range m.Modules[state.Component].Entries {
			work.Comparisons++
			if step.Operation == name {
				found = true
				break
			}
		}
		if !found {
			return work, fmt.Errorf("entry not exported")
		}
	}
	return work, nil
}
func Aliases(m Manifest) []string {
	out := []string{}
	for k := range m.Modules {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
