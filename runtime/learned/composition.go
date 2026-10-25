package learned

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/LynnColeArt/phlam/compiler/contracts"
	"github.com/LynnColeArt/phlam/semantics"
)

const CompositionABI = "phlam.counter.finite-domain-composition.v0"

type Operation struct {
	ID    string `json:"operation"`
	Delta int64  `json:"delta"`
}
type Sequence struct {
	ID      string      `json:"id"`
	Split   string      `json:"split"`
	Initial int64       `json:"initial"`
	Steps   []Operation `json:"steps"`
}
type Prediction struct {
	Next   int64 `json:"next"`
	Result int64 `json:"result"`
	Write  bool  `json:"write"`
}
type CompositionWork struct {
	Relation         contracts.Work `json:"relation"`
	DomainChecks     uint64         `json:"domain_checks"`
	PredictionChecks uint64         `json:"prediction_checks"`
	CertificateUses  uint64         `json:"certificate_uses"`
}
type Certificate struct {
	ABI         string         `json:"abi"`
	Key         string         `json:"key"`
	Valid       bool           `json:"valid"`
	Cases       int            `json:"primitive_cases_checked"`
	BuildNS     int64          `json:"build_ns"`
	Relation    contracts.Work `json:"relation_work"`
	FailureCase string         `json:"failure_case,omitempty"`
}
type CompositionOutcome struct {
	ID         string          `json:"id"`
	Accepted   bool            `json:"accepted"`
	Rejection  string          `json:"rejection,omitempty"`
	FailedStep int             `json:"failed_step"`
	State      int64           `json:"state"`
	Version    uint64          `json:"version"`
	Results    []int64         `json:"results,omitempty"`
	Writes     []int64         `json:"writes,omitempty"`
	Work       CompositionWork `json:"checker_work"`
	Timing     Timing          `json:"timing"`
}

// Composer seals an immutable executor/engine pair. A certificate is an in-process
// result of exhaustive exact checking, never a caller-supplied approval token.
// It saves no transition table and still computes every learned prediction.
type Composer struct {
	ready       bool
	executor    Executor
	engine      string
	boundary    string
	certificate Certificate
}

func addWork(a *contracts.Work, b contracts.Work) {
	a.TermEvaluations += b.TermEvaluations
	a.IntegerPrimitives += b.IntegerPrimitives
	a.ArgumentChecks += b.ArgumentChecks
	a.FieldChecks += b.FieldChecks
	a.OrderedWriteChecks += b.OrderedWriteChecks
}
func (e *Executor) NewComposer(engine, boundary string) (*Composer, Certificate, error) {
	if engine != "learned" && engine != "lookup" && engine != "zero" && engine != "shuffled" {
		return nil, Certificate{}, fmt.Errorf("unknown engine")
	}
	if boundary != "concrete" && boundary != "certified" {
		return nil, Certificate{}, fmt.Errorf("unknown boundary")
	}
	start := time.Now()
	copy := *e
	raw, err := json.Marshal(e.artifact)
	if err != nil {
		return nil, Certificate{}, err
	}
	var sealed Artifact
	if err := json.Unmarshal(raw, &sealed); err != nil {
		return nil, Certificate{}, err
	}
	copy.artifact = sealed
	payload, err := json.Marshal(struct {
		Artifact    Artifact
		Engine, ABI string
	}{copy.artifact, engine, CompositionABI})
	if err != nil {
		return nil, Certificate{}, err
	}
	p := &Composer{ready: true, executor: copy, engine: engine, boundary: boundary, certificate: Certificate{ABI: CompositionABI, Key: Hash(payload)}}
	if boundary == "concrete" {
		p.certificate.BuildNS = time.Since(start).Nanoseconds()
		return p, p.certificate, nil
	}
	// Enumerate primitive inputs, not compositions. Rechecking on this worker's
	// actual float64 implementation avoids trusting a persisted proof bit across
	// binaries or machines. The resulting proof flag belongs only to this pair.
	for s := int64(-64); s <= 64; s++ {
		for _, id := range []string{"Counter.read", "Counter.increment"} {
			lo, hi := int64(-16), int64(16)
			if id == "Counter.read" {
				lo, hi = 0, 0
			}
			for d := lo; d <= hi; d++ {
				c := Case{ID: fmt.Sprintf("%s:%d:%d", id, s, d), Operation: id, State: s, Delta: d}
				outcome := copy.Run(c, engine)
				p.certificate.Cases++
				addWork(&p.certificate.Relation, outcome.CheckerWork)
				if !outcome.Accepted {
					p.certificate.FailureCase = c.ID
					p.certificate.BuildNS = time.Since(start).Nanoseconds()
					return nil, p.certificate, fmt.Errorf("finite-domain certification failed at %s", c.ID)
				}
			}
		}
	}
	p.certificate.Valid = true
	p.certificate.BuildNS = time.Since(start).Nanoseconds()
	return p, p.certificate, nil
}
func (p *Composer) scores(c Case) [3]float64 {
	scores, _ := p.executor.predict(c, p.engine)
	return scores
}

func (p *Composer) Run(s Sequence) (out CompositionOutcome) {
	start := time.Now()
	out = CompositionOutcome{ID: s.ID, FailedStep: -1, State: s.Initial, Version: 1}
	defer func() {
		out.Timing.TotalNS = time.Since(start).Nanoseconds()
		out.Timing.OverheadNS = out.Timing.TotalNS - out.Timing.ProposalNS - out.Timing.CandidateNS - out.Timing.CheckerNS - out.Timing.CommitNS
	}()
	t := time.Now()
	valid := p.ready && len(s.Steps) >= 1 && len(s.Steps) <= 16
	out.Work.DomainChecks++
	for _, op := range s.Steps {
		out.Work.DomainChecks++
		if op.ID != "Counter.read" && op.ID != "Counter.increment" || op.Delta < -16 || op.Delta > 16 || op.ID == "Counter.read" && op.Delta != 0 {
			valid = false
		}
	}
	if p.boundary == "certified" && !p.certificate.Valid {
		valid = false
	}
	out.Timing.CheckerNS += time.Since(t).Nanoseconds()
	if !valid {
		out.Rejection = "invalid_sequence"
		return
	}
	t = time.Now()
	predictions := make([]Prediction, 0, len(s.Steps))
	state := s.Initial
	out.Timing.CandidateNS += time.Since(t).Nanoseconds()
	for i, op := range s.Steps {
		t = time.Now()
		out.Work.DomainChecks++
		inDomain := state >= -64 && state <= 64
		out.Timing.CheckerNS += time.Since(t).Nanoseconds()
		if !inDomain {
			out.Rejection = "outside_certified_domain"
			out.FailedStep = i
			return
		}
		t = time.Now()
		scores := p.scores(Case{Operation: op.ID, State: state, Delta: op.Delta})
		out.Timing.ProposalNS += time.Since(t).Nanoseconds()
		t = time.Now()
		valid := true
		for _, score := range scores {
			out.Work.PredictionChecks++
			if math.IsNaN(score) || math.IsInf(score, 0) || math.Abs(score) > 4 {
				valid = false
			}
		}
		out.Timing.CheckerNS += time.Since(t).Nanoseconds()
		if !valid {
			out.Rejection = "invalid_prediction"
			out.FailedStep = i
			return
		}
		t = time.Now()
		prediction := decodePrediction(scores)
		predictions = append(predictions, prediction)
		state = prediction.Next
		out.Timing.CandidateNS += time.Since(t).Nanoseconds()
	}
	t = time.Now()
	if p.boundary == "concrete" {
		state = s.Initial
		for i, op := range s.Steps {
			var contract contracts.Contract
			for _, c := range p.executor.artifact.Contracts.Methods {
				if c.ID == op.ID {
					contract = c
				}
			}
			prediction := predictions[i]
			args := map[string]semantics.Value{}
			if op.ID == "Counter.increment" {
				args["delta"] = semantics.Integer(op.Delta)
			}
			writes := []contracts.Write{}
			if prediction.Write {
				writes = append(writes, contracts.Write{Slot: "value", Value: semantics.Integer(prediction.Next)})
			}
			transition := contracts.Transition{Before: map[string]semantics.Value{"value": semantics.Integer(state)}, After: map[string]semantics.Value{"value": semantics.Integer(prediction.Next)}, Args: args, Result: semantics.Integer(prediction.Result), Writes: writes}
			work, err := contracts.ValidateMeasured(contract, transition)
			addWork(&out.Work.Relation, work)
			out.Timing.CheckerCalls++
			if err != nil {
				out.Timing.CheckerNS += time.Since(t).Nanoseconds()
				out.Rejection = "contract_rejected"
				out.FailedStep = i
				return
			}
			state = prediction.Next
		}
	} else {
		out.Work.CertificateUses = uint64(len(s.Steps))
	}
	out.Timing.CheckerNS += time.Since(t).Nanoseconds()
	t = time.Now()
	results := make([]int64, 0, len(predictions))
	writes := []int64{}
	for _, prediction := range predictions {
		results = append(results, prediction.Result)
		if prediction.Write {
			writes = append(writes, prediction.Next)
		}
	}
	out.Accepted = true
	out.State = state
	out.Results = results
	out.Writes = writes
	if len(writes) > 0 {
		out.Version = 2
	}
	out.Timing.CommitNS = time.Since(t).Nanoseconds()
	return
}
