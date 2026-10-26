package build

import (
	"fmt"
	abi "github.com/LynnColeArt/phlam/runtime/components"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func readJSON(path string, out any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, 256*1024+1))
	if err != nil {
		return err
	}
	if len(raw) > 256*1024 {
		return fmt.Errorf("%s: input size limit", path)
	}
	if err := abi.StrictJSON(raw, out); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
func local(root, name string) (string, error) {
	if name == "" || filepath.IsAbs(name) || filepath.Clean(name) != name || name == "." || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe relative input %q", name)
	}
	path := filepath.Join(root, name)
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, real)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("input escapes project: %s", name)
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("input is not regular: %s", name)
	}
	return real, nil
}
func inputs(path string) (Manifest, Plan, map[string][]byte, error) {
	var m Manifest
	var p Plan
	absolute, err := filepath.Abs(path)
	if err != nil {
		return m, p, nil, err
	}
	root, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil {
		return m, p, nil, err
	}
	if err := readJSON(absolute, &m); err != nil {
		return m, p, nil, err
	}
	if m.Schema != "phlame.build.v0" || m.Module.Name == "" || m.Module.Version == "" || len(m.Sources) < 1 || len(m.Sources) > 8 || len(m.Components) != len(m.Sources) {
		return m, p, nil, fmt.Errorf("%s: invalid manifest/schema", path)
	}
	planPath, err := local(root, m.TrainingPlan)
	if err != nil {
		return m, p, nil, err
	}
	if err := readJSON(planPath, &p); err != nil {
		return m, p, nil, err
	}
	if p.Schema != "phlame.training-plan.v0" || p.Sampling.StateStride < 1 || p.Sampling.StateStride > 128 || p.Sampling.ArgStride < 1 || p.Sampling.ArgStride > 32 || p.MaxCases < 1 || p.MaxCases > 50000 || p.CandidateTimeoutMS < 1 || p.CandidateTimeoutMS > 600000 || p.BuildTimeoutMS < 1 || p.BuildTimeoutMS > 7200000 || len(p.Candidates) < 1 || len(p.Candidates) > 16 {
		return m, p, nil, fmt.Errorf("%s: invalid training plan/budgets", m.TrainingPlan)
	}
	attempts := 0
	for _, r := range p.Candidates {
		if r.Kind != "affine" && r.Kind != "relu" || len(r.Seeds) < 1 || len(r.Seeds) > 16 {
			return m, p, nil, fmt.Errorf("unknown recipe or unbounded seeds")
		}
		if r.Kind == "affine" && (r.Width != 0 || r.Initialization != "legacy") || r.Kind == "relu" && (r.Width < 1 || r.Width > 32 || r.Initialization != "legacy" && r.Initialization != "paired-centered" && r.Initialization != "paired-affine") {
			return m, p, nil, fmt.Errorf("invalid recipe shape/initialization")
		}
		attempts += len(r.Seeds)
	}
	if attempts > 32 {
		return m, p, nil, fmt.Errorf("candidate budget exceeds 32")
	}
	lockPath, err := local(root, m.Lock)
	if err != nil {
		return m, p, nil, err
	}
	var lock Lock
	if err := readJSON(lockPath, &lock); err != nil {
		return m, p, nil, err
	}
	if lock.Schema != "phlame.lock.v0" {
		return m, p, nil, fmt.Errorf("unknown lock schema")
	}
	data := map[string][]byte{}
	names := append(append([]string{}, m.Sources...), m.TrainingPlan)
	for _, name := range names {
		if _, duplicate := data[name]; duplicate {
			return m, p, nil, fmt.Errorf("duplicate locked input %s", name)
		}
		file, err := local(root, name)
		if err != nil {
			return m, p, nil, err
		}
		handle, err := os.Open(file)
		if err != nil {
			return m, p, nil, err
		}
		raw, err := io.ReadAll(io.LimitReader(handle, 128*1024+1))
		closeErr := handle.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return m, p, nil, err
		}
		if len(raw) > 128*1024 {
			return m, p, nil, fmt.Errorf("%s: source/plan size limit", name)
		}
		if lock.Files[name] != abi.Hash(raw) {
			return m, p, nil, fmt.Errorf("%s: missing or mismatched lock hash", name)
		}
		data[name] = raw
	}
	if len(lock.Files) != len(data) {
		return m, p, nil, fmt.Errorf("lock contains undeclared inputs")
	}
	sourceNames := map[string]bool{}
	for _, name := range m.Sources {
		sourceNames[name] = true
	}
	aliases := map[string]bool{}
	used := map[string]bool{}
	for _, c := range m.Components {
		if c.Alias == "" || strings.ContainsAny(c.Alias, "/\\.") || aliases[c.Alias] || !sourceNames[c.Source] || used[c.Source] || c.Class == "" || len(c.Entries) < 1 {
			return m, p, nil, fmt.Errorf("invalid/duplicate component or undeclared source")
		}
		aliases[c.Alias] = true
		used[c.Source] = true
		d := c.Domain
		if d.StateMin < -64 || d.StateMax > 64 || d.StateMin > d.StateMax || d.ArgMin < -16 || d.ArgMax > 16 || d.ArgMin > d.ArgMax {
			return m, p, nil, fmt.Errorf("%s: invalid specialization domain", c.Source)
		}
	}
	return m, p, data, nil
}
