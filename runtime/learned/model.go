// Package learned implements the first bounded Counter experiment. Learned
// numeric proposals are distinct from the exact checker and local commit step.
// There is no source loader or reference-evaluator fallback in this package.
package learned

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/LynnColeArt/phlam/compiler/contracts"
	"github.com/LynnColeArt/phlam/semantics"
)

const Scale = 64.0
const Epochs = 8000
const Rate = 0.3

type Case struct {
	ID        string `json:"id"`
	Split     string `json:"split"`
	Operation string `json:"operation"`
	State     int64  `json:"state"`
	Delta     int64  `json:"delta"`
}
type Label struct {
	Case      Case   `json:"case"`
	Next      int64  `json:"next"`
	Result    int64  `json:"result"`
	Write     bool   `json:"write"`
	TraceHash string `json:"observed_trace_sha256"`
}
type Model struct {
	Weights [3][4]float64 `json:"weights"`
}

func features(c Case) [4]float64 {
	op := 0.0
	if c.Operation == "Counter.increment" {
		op = 1
	}
	return [4]float64{1, float64(c.State) / Scale, float64(c.Delta) / Scale, op}
}
func (m Model) Predict(c Case) (out [3]float64) {
	x := features(c)
	for h := range out {
		for j, v := range x {
			out[h] += m.Weights[h][j] * v
		}
	}
	return out
}

// Train uses full-batch gradient descent with a deterministic PRNG. Labels must
// come from real oracle runs, never from this model or the contract evaluator.
func Train(labels []Label, seed int64) (Model, float64, float64, error) {
	if len(labels) == 0 {
		return Model{}, 0, 0, fmt.Errorf("empty training set")
	}
	random := rand.New(rand.NewSource(seed))
	var m Model
	for h := range m.Weights {
		for j := range m.Weights[h] {
			m.Weights[h][j] = (random.Float64() - .5) * .1
		}
	}
	loss := func() float64 {
		sum := 0.0
		for _, l := range labels {
			p := m.Predict(l.Case)
			write := 0.0
			if l.Write {
				write = 1
			}
			target := [3]float64{float64(l.Next) / Scale, float64(l.Result) / Scale, write}
			for h := range p {
				d := p[h] - target[h]
				sum += d * d
			}
		}
		return sum / float64(3*len(labels))
	}
	initial := loss()
	for epoch := 0; epoch < Epochs; epoch++ {
		var gradient [3][4]float64
		for _, l := range labels {
			x := features(l.Case)
			p := m.Predict(l.Case)
			write := 0.0
			if l.Write {
				write = 1
			}
			target := [3]float64{float64(l.Next) / Scale, float64(l.Result) / Scale, write}
			for h := range p {
				err := p[h] - target[h]
				for j, v := range x {
					gradient[h][j] += err * v
				}
			}
		}
		for h := range m.Weights {
			for j := range m.Weights[h] {
				m.Weights[h][j] -= Rate * 2 * gradient[h][j] / float64(len(labels))
			}
		}
	}
	return m, initial, loss(), nil
}

type Timing struct {
	OverheadNS   int64 `json:"overhead_ns"`
	ProposalNS   int64 `json:"proposal_ns"`
	CandidateNS  int64 `json:"candidate_ns"`
	CheckerNS    int64 `json:"checker_ns"`
	CommitNS     int64 `json:"commit_ns"`
	TotalNS      int64 `json:"total_ns"`
	CheckerCalls int   `json:"checker_calls"`
}
type Outcome struct {
	CheckerWork contracts.Work `json:"checker_work"`
	ID          string         `json:"id"`
	Accepted    bool           `json:"accepted"`
	Rejection   string         `json:"rejection,omitempty"`
	State       int64          `json:"state"`
	Result      int64          `json:"result"`
	Version     uint64         `json:"version"`
	Write       bool           `json:"write"`
	Scores      [3]float64     `json:"scores"`
	Timing      Timing         `json:"timing"`
}

// Executor owns a loaded immutable artifact. Each Run starts from a fresh local
// version-1 state; returned version 2 denotes a checked write, 1 a read/rejection.
// This is a transition experiment, not an authenticated persistent-object server.
var lookupWrites = map[string]bool{"Counter.read": false, "Counter.increment": true}

type Executor struct{ artifact Artifact }

func (e *Executor) Run(c Case, engine string) (out Outcome) {
	start := time.Now()
	out = Outcome{ID: c.ID, State: c.State, Version: 1}
	defer func() {
		out.Timing.TotalNS = time.Since(start).Nanoseconds()
		out.Timing.OverheadNS = out.Timing.TotalNS - out.Timing.ProposalNS - out.Timing.CandidateNS - out.Timing.CheckerNS - out.Timing.CommitNS
	}()
	if c.Operation != "Counter.increment" && c.Operation != "Counter.read" {
		out.Rejection = "unknown_operation"
		return
	}
	if c.State < -64 || c.State > 64 || c.Delta < -16 || c.Delta > 16 || c.Operation == "Counter.read" && c.Delta != 0 {
		out.Rejection = "outside_domain"
		return
	}
	var contract contracts.Contract
	for _, candidate := range e.artifact.Contracts.Methods {
		if candidate.ID == c.Operation {
			contract = candidate
		}
	}
	t := time.Now()
	switch engine {
	case "learned":
		out.Scores = e.artifact.Model.Predict(c)
	case "zero":
		out.Scores = Model{}.Predict(c)
	case "shuffled":
		// Fixed permutation preserves the weight distribution but breaks its feature
		// and output associations. It is an ablation, not a retrained baseline.
		var m Model
		for h := range m.Weights {
			for j := range m.Weights[h] {
				m.Weights[h][j] = e.artifact.Model.Weights[(h+1)%3][(j+1)%4]
			}
		}
		out.Scores = m.Predict(c)
	case "lookup":
		// Structural baseline: exact operation lookup + exact primitive, not a memo
		// table of training cases. This is deliberately a strong trivial baseline.
		value := c.State
		write := 0.0
		if lookupWrites[c.Operation] {
			value, _ = semantics.Add(value, c.Delta)
			write = 1
		}
		out.Scores = [3]float64{float64(value) / Scale, float64(value) / Scale, write}
	default:
		out.Rejection = "unknown_engine"
		return
	}
	out.Timing.ProposalNS = time.Since(t).Nanoseconds()
	t = time.Now()
	for _, score := range out.Scores {
		if math.IsNaN(score) || math.IsInf(score, 0) || math.Abs(score) > 4 {
			out.Rejection = "invalid_prediction"
			return
		}
	}
	next := int64(math.Round(out.Scores[0] * Scale))
	result := int64(math.Round(out.Scores[1] * Scale))
	write := out.Scores[2] >= .5
	before := map[string]semantics.Value{"value": semantics.Integer(c.State)}
	after := map[string]semantics.Value{"value": semantics.Integer(next)}
	args := map[string]semantics.Value{}
	if c.Operation == "Counter.increment" {
		args["delta"] = semantics.Integer(c.Delta)
	}
	writes := []contracts.Write{}
	if write {
		writes = append(writes, contracts.Write{Slot: "value", Value: semantics.Integer(next)})
	}
	candidate := contracts.Transition{Before: before, After: after, Args: args, Result: semantics.Integer(result), Writes: writes}
	out.Timing.CandidateNS = time.Since(t).Nanoseconds()
	t = time.Now()
	work, err := contracts.ValidateMeasured(contract, candidate)
	out.CheckerWork = work
	out.Timing.CheckerNS = time.Since(t).Nanoseconds()
	out.Timing.CheckerCalls = 1
	if err != nil {
		out.Rejection = "contract_rejected"
		return
	}
	t = time.Now()
	out.Accepted = true
	out.State = next
	out.Result = result
	out.Write = write
	if write {
		out.Version++
	}
	out.Timing.CommitNS = time.Since(t).Nanoseconds()
	return
}
