package components

import (
	"encoding/json"
	"fmt"
	"time"
)

type Interval struct{ Low, High int64 }
type Certificate struct {
	Key            string `json:"key"`
	ModuleHash     string `json:"module_sha256"`
	Certified      bool   `json:"certified"`
	Cases          int    `json:"cases"`
	AcceptedInputs int    `json:"accepted_inputs"`
	RejectedInputs int    `json:"contract_rejected_inputs"`
	BuildNS        int64  `json:"build_ns"`
	Work           Work   `json:"checker_work"`
	Failure        string `json:"failure,omitempty"`
}
type module struct {
	artifact    Artifact
	root        string
	certificate Certificate
	admission   []map[int64][]Interval
}
type Registry struct {
	ready            bool
	engine, boundary string
	modules          map[string]*module
}

func Root(a Artifact) string { data, _ := json.Marshal(a); return Hash(data) }
func predict(a Artifact, index int, state, arg int64, engine string) (Prediction, error) {
	switch engine {
	case "learned":
		return Decode(a.Model.Predict(len(a.Contract.Operations), index, state, arg))
	case "zero":
		m := Model{Weights: make([][]float64, 3)}
		for i := range m.Weights {
			m.Weights[i] = make([]float64, 3+len(a.Contract.Operations))
		}
		return Decode(m.Predict(len(a.Contract.Operations), index, state, arg))
	case "shuffled":
		m := Model{Weights: make([][]float64, 3)}
		for h := range m.Weights {
			m.Weights[h] = make([]float64, len(a.Model.Weights[h]))
			for j := range m.Weights[h] {
				m.Weights[h][j] = a.Model.Weights[(h+1)%3][(j+1)%len(m.Weights[h])]
			}
		}
		return Decode(m.Predict(len(a.Contract.Operations), index, state, arg))
	case "lookup":
		var work Work
		expected, err := a.Contract.Operations[index].Evaluate(state, arg, &work)
		if err != nil || !expected.Accepted {
			return Prediction{}, fmt.Errorf("lookup rejected")
		}
		return expected.Prediction, nil
	}
	return Prediction{}, fmt.Errorf("unknown engine")
}

// NewRegistry seals each independent artifact. Certificates and admission rows
// belong to its module root/engine/ABI, never to another module or caller token.
func NewRegistry(artifacts map[string]Artifact, engine, boundary string) (*Registry, map[string]Certificate, error) {
	if len(artifacts) == 0 || len(artifacts) > 8 {
		return nil, nil, fmt.Errorf("module registry bounds")
	}
	if engine != "learned" && engine != "lookup" && engine != "zero" && engine != "shuffled" {
		return nil, nil, fmt.Errorf("unknown engine")
	}
	if boundary != "concrete" && boundary != "certified" {
		return nil, nil, fmt.Errorf("unknown boundary")
	}
	r := &Registry{ready: true, engine: engine, boundary: boundary, modules: map[string]*module{}}
	certificates := map[string]Certificate{}
	// Stable aliases determine processing order only; model inputs never contain
	// these aliases, another module's state, or another model's parameters.
	aliases := sortedKeys(artifacts)
	for _, alias := range aliases {
		start := time.Now()
		raw, err := json.Marshal(artifacts[alias])
		if err != nil {
			return nil, certificates, err
		}
		var a Artifact
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, certificates, err
		}
		if err := ValidateArtifact(a); err != nil {
			return nil, certificates, err
		}
		root := Hash(raw)
		keyData, _ := json.Marshal(struct{ Root, Engine, ABI string }{root, engine, ABI})
		m := &module{artifact: a, root: root, certificate: Certificate{Key: Hash(keyData), ModuleHash: root}, admission: make([]map[int64][]Interval, len(a.Contract.Operations))}
		if boundary == "certified" {
			for index, op := range a.Contract.Operations {
				m.admission[index] = map[int64][]Interval{}
				for state := a.Domain.StateMin; state <= a.Domain.StateMax; state++ {
					lo, hi := a.Domain.ArgMin, a.Domain.ArgMax
					if op.Arity == 0 {
						lo, hi = 0, 0
					}
					for arg := lo; arg <= hi; arg++ {
						m.certificate.Cases++
						var work Work
						expected, err := op.Evaluate(state, arg, &work)
						AddWork(&m.certificate.Work, work)
						if err == nil && expected.Accepted {
							m.certificate.AcceptedInputs++
							candidate, predictErr := predict(a, index, state, arg, engine)
							if predictErr != nil || candidate != expected.Prediction {
								err = fmt.Errorf("model differs from contract at operation %s state %d arg %d", op.Name, state, arg)
							}
							m.certificate.Work.Comparisons += 3
							if err == nil {
								rows := m.admission[index][state]
								if len(rows) > 0 && rows[len(rows)-1].High+1 == arg {
									rows[len(rows)-1].High = arg
								} else {
									rows = append(rows, Interval{arg, arg})
								}
								m.admission[index][state] = rows
							}
						} else if err == nil {
							m.certificate.RejectedInputs++
						}
						if err != nil {
							m.certificate.Failure = err.Error()
							m.certificate.BuildNS = time.Since(start).Nanoseconds()
							certificates[alias] = m.certificate
							return nil, certificates, err
						}
					}
				}
			}
			m.certificate.Certified = true
		}
		m.certificate.BuildNS = time.Since(start).Nanoseconds()
		r.modules[alias] = m
		certificates[alias] = m.certificate
	}
	return r, certificates, nil
}
