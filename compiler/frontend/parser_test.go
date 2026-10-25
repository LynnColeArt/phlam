package frontend

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyntaxCorpus(t *testing.T) {
	for _, kind := range []string{"valid", "invalid"} {
		files, err := filepath.Glob("../../tests/syntax/" + kind + "/*.phlam")
		if err != nil || len(files) == 0 {
			t.Fatal("missing corpus", err)
		}
		for _, path := range files {
			t.Run(filepath.Base(path), func(t *testing.T) {
				src, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				_, err = Parse(string(src))
				if kind == "valid" {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				if err == nil {
					t.Fatal("invalid source accepted")
				}
				want, readErr := os.ReadFile(strings.TrimSuffix(path, ".phlam") + ".error")
				if readErr != nil {
					t.Fatal(readErr)
				}
				var d *Diagnostic
				if !errors.As(err, &d) || d.Code != strings.TrimSpace(string(want)) {
					t.Fatalf("want %s, got %v", want, err)
				}
				if d.Span.Line < 1 || d.Span.Column < 1 || d.Span.Start < 0 || d.Span.End > len(src) {
					t.Fatalf("invalid diagnostic span: %+v", d)
				}
			})
		}
	}
}
func TestCounterStructureAndProvenance(t *testing.T) {
	src, err := os.ReadFile("../../examples/counter/counter.phlam")
	if err != nil {
		t.Fatal(err)
	}
	p, err := Parse(string(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Classes) != 1 || p.Classes[0].Name != "Counter" || !strings.Contains(p.Classes[0].Text, "independent") && !strings.Contains(p.Classes[0].Text, "separate state") {
		t.Fatal("class contract missing")
	}
	c := p.Classes[0]
	if len(c.Children) != 4 {
		t.Fatal("wrong member count")
	}
	increment := c.Children[2]
	if increment.Kind != "method" || increment.Name != "increment" {
		t.Fatal("missing increment")
	}
	body := increment.Children[1]
	if body.Children[0].Kind != "assign" || body.Children[0].Text != "+=" || body.Children[1].Kind != "return" {
		t.Fatal("wrong increment structure")
	}
	ids := map[string]bool{}
	var visit func(*Node)
	visit = func(n *Node) {
		if ids[n.ID] {
			t.Fatal("duplicate provenance ID", n.ID)
		}
		ids[n.ID] = true
		if n.Span.Start < 0 || n.Span.End > len(src) || n.Span.End <= n.Span.Start {
			t.Fatalf("bad span: %+v", n)
		}
		for _, child := range n.Children {
			visit(child)
		}
	}
	visit(c)
	a, _ := json.Marshal(p)
	p2, err := Parse(string(src))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(p2)
	if string(a) != string(b) {
		t.Fatal("nondeterministic output")
	}
	// JSON must retain integers as source text, avoiding float64 precision loss.
	big, err := Parse("/// Numbers.\nclass Numbers {$value = 9223372036854775807}")
	if err != nil || big.Classes[0].Children[0].Children[0].Text != "9223372036854775807" {
		t.Fatal("integer literal corrupted", err)
	}
}
func TestNestingLimits(t *testing.T) {
	for _, expr := range []string{strings.Repeat("(", 300) + "1" + strings.Repeat(")", 300), strings.Repeat("if (true) {", 140) + "return 1;" + strings.Repeat("}", 140)} {
		_, err := Parse("/// Bounded.\nclass Bounded {function run() {" + expr + "}}")
		var d *Diagnostic
		if !errors.As(err, &d) || d.Code != "P116" {
			t.Fatalf("want nesting diagnostic, got %v", err)
		}
	}
}
func FuzzParse(f *testing.F) {
	for _, s := range []string{"", "/// Counter.\nclass Counter {$value=0}", "/// C.\nclass C {function f(){return (1\n+2)}}", "/*\n", "$", "\xff"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 65536 {
			t.Skip()
		}
		_, _ = Parse(s)
	})
}

func TestExpressionBindingAndUniqueIDs(t *testing.T) {
	src := "/// Binding.\nclass Binding {function run() {\n$x = 1+2+3\n$$name()\n$obj->a->b\nreturn (1+2)*3\n}}"
	p, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	body := p.Classes[0].Children[0].Children[0]
	call := body.Children[1].Children[0]
	if call.Kind != "call" || call.Children[0].Kind != "dynamic_variable" {
		t.Fatalf("wrong indirect call binding: %+v", call)
	}
	result := body.Children[3].Children[0]
	if result.Text != "*" || result.Children[0].Text != "+" {
		t.Fatal("wrong expression precedence")
	}
	seen := map[string]bool{}
	var walk func(*Node)
	walk = func(n *Node) {
		if seen[n.ID] {
			t.Fatal("duplicate ID", n.ID)
		}
		seen[n.ID] = true
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(p.Classes[0])
}
