package components

import (
	"sort"
	"time"
)

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

type Binding struct {
	Literal *int64 `json:"literal,omitempty"`
	From    *int   `json:"from_result,omitempty"`
}
type Instance struct {
	Component string `json:"component"`
	Value     int64  `json:"value"`
	Version   uint64 `json:"version"`
}
type Step struct {
	Target    string   `json:"target"`
	Operation string   `json:"operation"`
	Argument  *Binding `json:"argument,omitempty"`
}
type Transaction struct {
	ID        string              `json:"id"`
	Instances map[string]Instance `json:"instances"`
	Steps     []Step              `json:"steps"`
}
type Return struct {
	Type     string `json:"type"`
	Value    int64  `json:"value"`
	Producer string `json:"producer"`
}
type Write struct {
	Target string `json:"target"`
	Slot   string `json:"slot"`
	Value  int64  `json:"value"`
}
type Measurements struct {
	BindingChecks   uint64 `json:"binding_checks"`
	FrameChecks     uint64 `json:"frame_checks"`
	DomainChecks    uint64 `json:"domain_checks"`
	CertificateUses uint64 `json:"certificate_uses"`
	Relations       Work   `json:"relations"`
}
type Timing struct {
	ProposalNS int64 `json:"proposal_ns"`
	BindingNS  int64 `json:"binding_ns"`
	CheckerNS  int64 `json:"checker_ns"`
	CommitNS   int64 `json:"commit_ns"`
	OverheadNS int64 `json:"overhead_ns"`
	TotalNS    int64 `json:"total_ns"`
}
type Outcome struct {
	ID         string              `json:"id"`
	Accepted   bool                `json:"accepted"`
	Rejection  string              `json:"rejection,omitempty"`
	FailedStep int                 `json:"failed_step"`
	States     map[string]Instance `json:"states"`
	Results    []Return            `json:"results,omitempty"`
	Writes     []Write             `json:"writes,omitempty"`
	Work       Measurements        `json:"checker_work"`
	LookupWork Work                `json:"lookup_application_work"`
	Timing     Timing              `json:"timing"`
}

func cloneInstances(instances map[string]Instance) map[string]Instance {
	copy := make(map[string]Instance, len(instances))
	for k, v := range instances {
		copy[k] = v
	}
	return copy
}
func (r *Registry) Run(tx Transaction) (out Outcome) {
	start := time.Now()
	out = Outcome{ID: tx.ID, FailedStep: -1, States: cloneInstances(tx.Instances)}
	defer func() {
		out.Timing.TotalNS = time.Since(start).Nanoseconds()
		out.Timing.OverheadNS = out.Timing.TotalNS - out.Timing.BindingNS - out.Timing.ProposalNS - out.Timing.CheckerNS - out.Timing.CommitNS
	}()
	t := time.Now()
	valid := r != nil && r.ready && len(tx.Instances) > 0 && len(tx.Instances) <= 8 && len(tx.Steps) > 0 && len(tx.Steps) <= 16
	if valid {
		for _, state := range tx.Instances {
			out.Work.FrameChecks++
			m := r.modules[state.Component]
			if m == nil || state.Version == 0 || state.Version == ^uint64(0) {
				valid = false
				break
			}
		}
	}
	out.Timing.CheckerNS += time.Since(t).Nanoseconds()
	if !valid {
		out.Rejection = "invalid_transaction"
		return
	}
	working := cloneInstances(tx.Instances)
	results := make([]Return, 0, len(tx.Steps))
	writes := []Write{}
	dirty := map[string]bool{}
	for i, step := range tx.Steps {
		out.FailedStep = i
		t = time.Now()
		out.Work.BindingChecks++
		state, exists := working[step.Target]
		if !exists {
			out.Timing.BindingNS += time.Since(t).Nanoseconds()
			out.Rejection = "unknown_instance"
			return
		}
		m := r.modules[state.Component]
		index := -1
		for j, op := range m.artifact.Contract.Operations {
			if op.Name == step.Operation {
				index = j
			}
		}
		if index < 0 {
			out.Timing.BindingNS += time.Since(t).Nanoseconds()
			out.Rejection = "unknown_operation"
			return
		}
		op := m.artifact.Contract.Operations[index]
		arg := int64(0)
		bindingOK := true
		if op.Arity == 0 {
			bindingOK = step.Argument == nil
		} else if step.Argument == nil {
			bindingOK = false
		} else {
			b := step.Argument
			if (b.Literal == nil) == (b.From == nil) {
				bindingOK = false
			} else if b.Literal != nil {
				arg = *b.Literal
			} else {
				ref := *b.From
				if ref < 0 || ref >= len(results) || results[ref].Type != "int64" {
					bindingOK = false
				} else {
					arg = results[ref].Value
				}
			}
		}
		out.Timing.BindingNS += time.Since(t).Nanoseconds()
		if !bindingOK {
			out.Rejection = "invalid_value_binding"
			return
		}
		t = time.Now()
		out.Work.DomainChecks++
		d := m.artifact.Domain
		allowed := state.Value >= d.StateMin && state.Value <= d.StateMax && (op.Arity == 0 || arg >= d.ArgMin && arg <= d.ArgMax)
		if !allowed {
			out.Timing.CheckerNS += time.Since(t).Nanoseconds()
			out.Rejection = "outside_component_domain"
			return
		}
		var selected Expected
		if r.boundary == "certified" {
			allowed = false
			for _, interval := range m.admission[index][state.Value] {
				out.Work.DomainChecks++
				if arg >= interval.Low && arg <= interval.High {
					allowed = true
					break
				}
			}
			if !m.certificate.Certified {
				allowed = false
			}
		} else {
			var work Work
			expected, err := op.Evaluate(state.Value, arg, &work)
			AddWork(&out.Work.Relations, work)
			selected = expected
			allowed = err == nil && expected.Accepted
		}
		out.Timing.CheckerNS += time.Since(t).Nanoseconds()
		if !allowed {
			out.Rejection = "component_rejected"
			return
		}
		t = time.Now()
		var prediction Prediction
		var err error
		if r.engine == "lookup" {
			if r.boundary == "concrete" {
				prediction = selected.Prediction
			} else {
				var work Work
				selected, err = op.Evaluate(state.Value, arg, &work)
				AddWork(&out.LookupWork, work)
				prediction = selected.Prediction
			}
		} else {
			prediction, err = predict(m.artifact, index, state.Value, arg, r.engine)
		}
		out.Timing.ProposalNS += time.Since(t).Nanoseconds()
		if err != nil {
			out.Rejection = "invalid_prediction"
			return
		}
		t = time.Now()
		if r.boundary == "concrete" {
			out.Work.Relations.Comparisons += 3
			if prediction != selected.Prediction {
				out.Timing.CheckerNS += time.Since(t).Nanoseconds()
				out.Rejection = "prediction_mismatch"
				return
			}
		} else {
			out.Work.CertificateUses++
		}
		out.Timing.CheckerNS += time.Since(t).Nanoseconds()
		// Model outputs have no target, component, slot name, handle or capability.
		// Only this already-bound instance can be staged by this primitive.
		state.Value = prediction.Next
		working[step.Target] = state
		results = append(results, Return{Type: "int64", Value: prediction.Result, Producer: step.Target})
		if prediction.Write {
			dirty[step.Target] = true
			writes = append(writes, Write{Target: step.Target, Slot: m.artifact.Contract.Slot, Value: prediction.Next})
		}
	}
	t = time.Now()
	for target := range dirty {
		state := working[target]
		state.Version++
		working[target] = state
	}
	out.States = working
	out.Results = results
	out.Writes = writes
	out.Accepted = true
	out.FailedStep = -1
	out.Timing.CommitNS = time.Since(t).Nanoseconds()
	return
}
