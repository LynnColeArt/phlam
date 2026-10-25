// Package trace defines the recording seam for the future reference evaluator
// and learned executor. It does not execute an AST or generate training labels.
package trace

import "github.com/LynnColeArt/phlam/compiler/frontend"

type Event struct {
	Sequence     int           `json:"sequence"`
	Phase        string        `json:"phase"` // observed, proposed, accepted, rejected, committed
	NodeID       string        `json:"node_id"`
	Span         frontend.Span `json:"span"`
	StateVersion uint64        `json:"state_version"`
	Operation    string        `json:"operation"`
	// Values are deliberately absent until a redaction policy is implemented.
}
type Recorder struct{ events []Event }

func (r *Recorder) Record(e Event)  { e.Sequence = len(r.events); r.events = append(r.events, e) }
func (r *Recorder) Events() []Event { return append([]Event(nil), r.events...) }
