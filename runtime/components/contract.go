// Package components supplies the experimental scalar-component ABI. It has no
// component names, source parser, oracle, or cross-component arithmetic rules.
package components

import (
	"fmt"
	"github.com/LynnColeArt/phlam/semantics"
)

const ABI = "phlame.scalar-components.v0"

type Expr struct {
	Op    string `json:"op"`
	Int   int64  `json:"int,omitempty"`
	Bool  bool   `json:"bool,omitempty"`
	Left  *Expr  `json:"left,omitempty"`
	Right *Expr  `json:"right,omitempty"`
}
type Path struct {
	Guard  *Expr   `json:"guard"`
	Checks []*Expr `json:"checks,omitempty"`
	Post   *Expr   `json:"post"`
	Writes []*Expr `json:"writes,omitempty"`
	Return *Expr   `json:"return"`
}
type Operation struct {
	Name   string `json:"name"`
	Arity  int    `json:"arity"`
	NodeID string `json:"source_node_id"`
	Paths  []Path `json:"paths"`
}
type Contract struct {
	Class      string      `json:"class"`
	Slot       string      `json:"slot"`
	Purpose    string      `json:"purpose_declared"`
	SourceHash string      `json:"source_sha256"`
	Operations []Operation `json:"operations"`
}
type Domain struct {
	StateMin int64 `json:"state_min"`
	StateMax int64 `json:"state_max"`
	ArgMin   int64 `json:"arg_min"`
	ArgMax   int64 `json:"arg_max"`
}
type Work struct {
	Expressions uint64 `json:"expressions"`
	Arithmetic  uint64 `json:"integer_arithmetic"`
	Predicates  uint64 `json:"predicates"`
	Comparisons uint64 `json:"output_comparisons"`
}
type Prediction struct {
	Next   int64 `json:"next"`
	Result int64 `json:"result"`
	Write  bool  `json:"write"`
}
type Expected struct {
	Accepted   bool
	Prediction Prediction
}
type scalar struct {
	kind string
	i    int64
	b    bool
}

func (e *Expr) eval(state, arg int64, w *Work) (scalar, error) {
	if e == nil {
		return scalar{}, fmt.Errorf("missing contract expression")
	}
	w.Expressions++
	switch e.Op {
	case "slot":
		return scalar{kind: "int", i: state}, nil
	case "arg":
		return scalar{kind: "int", i: arg}, nil
	case "int":
		return scalar{kind: "int", i: e.Int}, nil
	case "bool":
		return scalar{kind: "bool", b: e.Bool}, nil
	}
	a, err := e.Left.eval(state, arg, w)
	if err != nil {
		return a, err
	}
	if e.Op == "not" {
		if a.kind != "bool" {
			return scalar{}, fmt.Errorf("boolean required")
		}
		w.Predicates++
		return scalar{kind: "bool", b: !a.b}, nil
	}
	if e.Op == "and" || e.Op == "or" {
		if a.kind != "bool" {
			return scalar{}, fmt.Errorf("boolean required")
		}
		w.Predicates++
		if e.Op == "and" && !a.b || e.Op == "or" && a.b {
			return a, nil
		}
	}
	b, err := e.Right.eval(state, arg, w)
	if err != nil {
		return b, err
	}
	if e.Op == "and" || e.Op == "or" {
		if b.kind != "bool" {
			return scalar{}, fmt.Errorf("boolean required")
		}
		return b, nil
	}
	if a.kind != "int" || b.kind != "int" {
		return scalar{}, fmt.Errorf("integer contract operands required")
	}
	switch e.Op {
	case "add", "sub", "mul":
		w.Arithmetic++
		var v int64
		switch e.Op {
		case "add":
			v, err = semantics.Add(a.i, b.i)
		case "sub":
			v, err = semantics.Sub(a.i, b.i)
		case "mul":
			v, err = semantics.Mul(a.i, b.i)
		}
		return scalar{kind: "int", i: v}, err
	}
	w.Predicates++
	result := false
	switch e.Op {
	case "lt":
		result = a.i < b.i
	case "gt":
		result = a.i > b.i
	case "le":
		result = a.i <= b.i
	case "ge":
		result = a.i >= b.i
	case "eq":
		result = a.i == b.i
	case "ne":
		result = a.i != b.i
	default:
		return scalar{}, fmt.Errorf("unknown expression opcode %s", e.Op)
	}
	return scalar{kind: "bool", b: result}, nil
}
func (op Operation) Evaluate(state, arg int64, w *Work) (Expected, error) {
	matched := false
	out := Expected{}
	for _, path := range op.Paths {
		guard, err := path.Guard.eval(state, arg, w)
		if err != nil {
			return out, err
		}
		if guard.kind != "bool" {
			return out, fmt.Errorf("nonboolean path guard")
		}
		if !guard.b {
			continue
		}
		if matched {
			return out, fmt.Errorf("overlapping contract paths")
		}
		matched = true
		for _, check := range path.Checks {
			if _, err := check.eval(state, arg, w); err != nil {
				return out, err
			}
		}
		post, err := path.Post.eval(state, arg, w)
		if err != nil || post.kind != "int" {
			return out, fmt.Errorf("invalid post state: %v", err)
		}
		result, err := path.Return.eval(state, arg, w)
		if err != nil {
			return out, err
		}
		if result.kind == "bool" && !result.b {
			if post.i != state || len(path.Writes) != 0 {
				return out, fmt.Errorf("rejection path has effects")
			}
			continue
		}
		if result.kind != "int" || len(path.Writes) > 1 {
			return out, fmt.Errorf("unsupported return or write history")
		}
		if len(path.Writes) == 0 && post.i != state {
			return out, fmt.Errorf("unwritten state changed")
		}
		out = Expected{Accepted: true, Prediction: Prediction{Next: post.i, Result: result.i, Write: len(path.Writes) == 1}}
		if len(path.Writes) == 1 {
			write, err := path.Writes[0].eval(state, arg, w)
			if err != nil || write.kind != "int" || write.i != post.i {
				return out, fmt.Errorf("invalid ordered write")
			}
		}
	}
	if !matched {
		return out, fmt.Errorf("nonexhaustive contract")
	}
	return out, nil
}
func (op Operation) Check(state, arg int64, p Prediction, w *Work) error {
	expected, err := op.Evaluate(state, arg, w)
	if err != nil {
		return err
	}
	w.Comparisons += 3
	if !expected.Accepted {
		return fmt.Errorf("contract rejects input")
	}
	if p != expected.Prediction {
		return fmt.Errorf("transition relation mismatch")
	}
	return nil
}
func AddWork(a *Work, b Work) {
	a.Expressions += b.Expressions
	a.Arithmetic += b.Arithmetic
	a.Predicates += b.Predicates
	a.Comparisons += b.Comparisons
}
