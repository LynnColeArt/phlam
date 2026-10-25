// Package trace records actual oracle execution and commit/failure observations.
// Integer values are captured only when explicitly enabled by a local caller.
// These traces are not a production authorization or redaction system.
package trace

import "github.com/LynnColeArt/phlam/compiler/frontend"

type Event struct {
	Transaction  uint64           `json:"transaction"`
	Sequence     int              `json:"sequence"`
	Phase        string           `json:"phase"`
	Invocation   uint64           `json:"invocation"`
	Parent       uint64           `json:"parent,omitempty"`
	NodeID       string           `json:"node_id"`
	Span         frontend.Span    `json:"span"`
	Object       uint64           `json:"object,omitempty"`
	StateVersion uint64           `json:"state_version"`
	Operation    string           `json:"operation"`
	ContractID   string           `json:"contract_id,omitempty"`
	Code         string           `json:"code,omitempty"`
	Integers     map[string]int64 `json:"integers,omitempty"`
}
type Recorder struct{ events []Event }

func copyEvent(e Event) Event {
	if e.Integers != nil {
		m := make(map[string]int64, len(e.Integers))
		for k, v := range e.Integers {
			m[k] = v
		}
		e.Integers = m
	}
	return e
}
func (r *Recorder) Record(e Event) {
	e.Sequence = len(r.events)
	r.events = append(r.events, copyEvent(e))
}
func (r *Recorder) Events() []Event {
	out := make([]Event, len(r.events))
	for i, e := range r.events {
		out[i] = copyEvent(e)
	}
	return out
}
