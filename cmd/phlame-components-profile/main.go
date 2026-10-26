// The profiling worker consumes frozen modules and oracle observations as data.
// It imports no source parser, compiler, training machinery or reference oracle.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/pprof"
	"time"

	abi "github.com/LynnColeArt/phlam/runtime/components"
)

type moduleRef struct {
	File string `json:"file"`
	Root string `json:"module_sha256"`
}
type manifest struct {
	ABI     string               `json:"abi"`
	Modules map[string]moduleRef `json:"modules"`
}
type observed struct {
	Transaction abi.Transaction         `json:"transaction"`
	Category    string                  `json:"category"`
	States      map[string]abi.Instance `json:"oracle_states"`
	Results     []abi.Return            `json:"oracle_results"`
	Writes      []abi.Write             `json:"oracle_writes"`
}
type behavior struct {
	ID         string                  `json:"id"`
	Accepted   bool                    `json:"accepted"`
	Rejection  string                  `json:"rejection,omitempty"`
	FailedStep int                     `json:"failed_step"`
	States     map[string]abi.Instance `json:"states"`
	Results    []abi.Return            `json:"results,omitempty"`
	Writes     []abi.Write             `json:"writes,omitempty"`
}
type output struct {
	Engine           string                     `json:"engine"`
	Boundary         string                     `json:"boundary"`
	Group            string                     `json:"group"`
	Timed            bool                       `json:"phase_clocks"`
	Iterations       int                        `json:"iterations"`
	GroupCases       int                        `json:"group_cases"`
	VerifiedCases    int                        `json:"verified_cases"`
	VerifiedAccepted int                        `json:"verified_accepted"`
	VerifiedRejected int                        `json:"verified_rejected"`
	BehaviorHash     string                     `json:"behavior_sha256"`
	LoadNS           int64                      `json:"load_ns"`
	Certificates     map[string]abi.Certificate `json:"certificates"`
	NS               float64                    `json:"ns_per_transaction"`
	Bytes            float64                    `json:"bytes_per_transaction"`
	Allocs           float64                    `json:"allocations_per_transaction"`
	GC               uint32                     `json:"gc_cycles"`
	Phases           abi.Timing                 `json:"total_phase_ns"`
	Work             abi.Measurements           `json:"total_checker_work"`
	LookupWork       abi.Work                   `json:"total_lookup_application_work"`
	CPUProfile       bool                       `json:"cpu_profile_enabled"`
}

var sink abi.Outcome

func load(path, keyPath string) (map[string]abi.Artifact, error) {
	key, err := abi.PublicKey(keyPath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) > 16384 {
		return nil, fmt.Errorf("manifest limit")
	}
	var m manifest
	if err := abi.StrictJSON(data, &m); err != nil {
		return nil, err
	}
	if m.ABI != abi.ABI || len(m.Modules) < 1 || len(m.Modules) > 8 {
		return nil, fmt.Errorf("invalid registry")
	}
	artifacts := map[string]abi.Artifact{}
	for alias, ref := range m.Modules {
		if alias == "" || filepath.Base(ref.File) != ref.File {
			return nil, fmt.Errorf("invalid module binding")
		}
		a, root, err := abi.Load(filepath.Join(filepath.Dir(path), ref.File), key)
		if err != nil {
			return nil, err
		}
		if root != ref.Root {
			return nil, fmt.Errorf("module root mismatch")
		}
		artifacts[alias] = a
	}
	return artifacts, nil
}
func run() error {
	registryPath := flag.String("registry", "registry.json", "signed module manifest")
	key := flag.String("key", "trusted-key.hex", "research public key")
	casesPath := flag.String("cases", "transactions.jsonl", "oracle observation data, not source")
	engine := flag.String("engine", "learned", "learned or lookup")
	boundary := flag.String("boundary", "certified", "concrete or certified")
	group := flag.String("group", "mixed", "mixed, accepted or scope_rejected")
	timed := flag.Bool("phase-clocks", false, "retain per-phase timestamps")
	iterations := flag.Int("iterations", 200000, "transactions to measure")
	cpu := flag.String("cpuprofile", "", "optional CPU profile output")
	mem := flag.String("memprofile", "", "optional allocation profile output")
	flag.Parse()
	if *iterations < 1 || *iterations > 10000000 || (*engine != "learned" && *engine != "lookup") || (*group != "mixed" && *group != "accepted" && *group != "scope_rejected") {
		return fmt.Errorf("invalid profile arguments")
	}
	started := time.Now()
	artifacts, err := load(*registryPath, *key)
	if err != nil {
		return err
	}
	o := output{Engine: *engine, Boundary: *boundary, Group: *group, Timed: *timed, Iterations: *iterations, CPUProfile: *cpu != "", LoadNS: time.Since(started).Nanoseconds()}
	r, certificates, err := abi.NewRegistry(artifacts, *engine, *boundary)
	if err != nil {
		return err
	}
	o.Certificates = certificates
	file, err := os.Open(*casesPath)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	cases := []abi.Transaction{}
	behaviors := []behavior{}
	for scanner.Scan() {
		var observed observed
		if err := json.Unmarshal(scanner.Bytes(), &observed); err != nil {
			return err
		}
		tx := observed.Transaction
		measured := r.Run(tx)
		unclocked := r.RunUninstrumented(tx)
		measured.Timing = abi.Timing{}
		if !reflect.DeepEqual(measured, unclocked) {
			return fmt.Errorf("clock modes change semantics: %s", tx.ID)
		}
		expected := observed.Category == "accepted"
		if measured.Accepted != expected {
			return fmt.Errorf("unexpected disposition: %s", tx.ID)
		}
		if expected {
			if !reflect.DeepEqual(measured.States, observed.States) || !reflect.DeepEqual(measured.Results, observed.Results) || !reflect.DeepEqual(measured.Writes, observed.Writes) {
				return fmt.Errorf("oracle mismatch: %s", tx.ID)
			}
			o.VerifiedAccepted++
		} else {
			if !reflect.DeepEqual(measured.States, tx.Instances) || len(measured.Results) != 0 || len(measured.Writes) != 0 {
				return fmt.Errorf("rollback violation: %s", tx.ID)
			}
			o.VerifiedRejected++
		}
		o.VerifiedCases++
		behaviors = append(behaviors, behavior{tx.ID, measured.Accepted, measured.Rejection, measured.FailedStep, measured.States, measured.Results, measured.Writes})
		if *group == "mixed" || observed.Category == *group || *group == "scope_rejected" && observed.Category == "outside_certified_scope" {
			cases = append(cases, tx)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(cases) == 0 {
		return fmt.Errorf("empty group")
	}
	raw, _ := json.Marshal(behaviors)
	o.BehaviorHash = abi.Hash(raw)
	o.GroupCases = len(cases)
	execute := r.RunUninstrumented
	if *timed {
		execute = r.Run
	}
	// Warm-up and validation are outside the measured/profiled interval.
	for i := 0; i < 10*len(cases); i++ {
		sink = execute(cases[i%len(cases)])
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	if *cpu != "" {
		f, err := os.Create(*cpu)
		if err != nil {
			return err
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			f.Close()
			return err
		}
		defer f.Close()
	}
	started = time.Now()
	for i := 0; i < *iterations; i++ {
		out := execute(cases[i%len(cases)])
		o.Phases.BindingNS += out.Timing.BindingNS
		o.Phases.ProposalNS += out.Timing.ProposalNS
		o.Phases.CheckerNS += out.Timing.CheckerNS
		o.Phases.CommitNS += out.Timing.CommitNS
		o.Phases.OverheadNS += out.Timing.OverheadNS
		o.Phases.TotalNS += out.Timing.TotalNS
		o.Work.BindingChecks += out.Work.BindingChecks
		o.Work.FrameChecks += out.Work.FrameChecks
		o.Work.DomainChecks += out.Work.DomainChecks
		o.Work.CertificateUses += out.Work.CertificateUses
		abi.AddWork(&o.Work.Relations, out.Work.Relations)
		abi.AddWork(&o.LookupWork, out.LookupWork)
		sink = out
	}
	elapsed := time.Since(started).Nanoseconds()
	if *cpu != "" {
		pprof.StopCPUProfile()
	}
	runtime.ReadMemStats(&after)
	o.NS = float64(elapsed) / float64(*iterations)
	o.Bytes = float64(after.TotalAlloc-before.TotalAlloc) / float64(*iterations)
	o.Allocs = float64(after.Mallocs-before.Mallocs) / float64(*iterations)
	o.GC = after.NumGC - before.NumGC
	if *mem != "" {
		runtime.GC()
		f, err := os.Create(*mem)
		if err != nil {
			return err
		}
		err = pprof.Lookup("allocs").WriteTo(f, 0)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return json.NewEncoder(os.Stdout).Encode(o)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
