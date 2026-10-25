// Package frontend parses the bootstrap PHLAM subset. Its AST is a compiler
// scaffold, not a deployable application artifact or an executable graph.
package frontend

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

type Span struct {
	Start  int `json:"start"`
	End    int `json:"end"`
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Node struct {
	ID       string  `json:"id"`
	Kind     string  `json:"kind"`
	Span     Span    `json:"span"`
	Name     string  `json:"name,omitempty"`
	Text     string  `json:"text,omitempty"`
	Children []*Node `json:"children,omitempty"`
}

type Program struct {
	Schema     string  `json:"schema"`
	SourceHash string  `json:"source_sha256"`
	Classes    []*Node `json:"classes"`
}

// IDs are stable for identical source bytes, independent of checkout location.
// They are version-local provenance IDs, not persistent public capability IDs.
func (p *parser) node(kind string, start token, children ...*Node) *Node {
	end := start.end
	if p.i > 0 && p.tokens[p.i-1].end > end {
		end = p.tokens[p.i-1].end
	}
	return &Node{ID: fmt.Sprintf("%s:%s:%d:%d", p.hash, kind, start.start, end), Kind: kind,
		Span: Span{start.start, end, start.line, start.column}, Children: children}
}

// Diagnostic keeps source byte spans as well as a readable location and hint.
type Diagnostic struct {
	Code    string
	Message string
	Hint    string
	Span    Span
}

func (d *Diagnostic) Error() string { return d.Code + ": " + d.Message }
func (d *Diagnostic) Render(path, source string) string {
	lines := strings.Split(source, "\n")
	line := ""
	if d.Span.Line > 0 && d.Span.Line <= len(lines) {
		line = strings.TrimSuffix(lines[d.Span.Line-1], "\r")
	}
	return fmt.Sprintf("error[%s]: %s\n --> %s:%d:%d\n %d | %s\n   | %s^\n help: %s\n", d.Code, d.Message, path, d.Span.Line, d.Span.Column, d.Span.Line, line, strings.Repeat(" ", max(0, d.Span.Column-1)), d.Hint)
}
func hashSource(s string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(s))) }
