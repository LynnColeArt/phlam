// Package contracts derives operational relations for a deliberately restricted
// straight-line integer fragment. Unknown methods stay unknown. Validation here
// evaluates relations for one concrete transition; it is not a symbolic proof of
// a learned term, and it does not execute or authorize model proposals.
package contracts

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/LynnColeArt/phlam/compiler/frontend"
	"github.com/LynnColeArt/phlam/semantics"
)

type Term struct {
	Op    string `json:"op"`
	Name  string `json:"name,omitempty"`
	Int   int64  `json:"int,omitempty"`
	Left  *Term  `json:"left,omitempty"`
	Right *Term  `json:"right,omitempty"`
}
type Update struct {
	Slot  string `json:"slot"`
	Value *Term  `json:"value"`
}
type Write struct {
	Slot  string          `json:"slot"`
	Value semantics.Value `json:"value"`
}
type Transition struct {
	Before, After, Args map[string]semantics.Value
	Result              semantics.Value
	Writes              []Write
}

type Parameter struct {
	Name       string `json:"name"`
	HasDefault bool   `json:"has_default"`
	Default    *int64 `json:"default,omitempty"`
}
type Contract struct {
	Domain     string           `json:"domain"`
	Updates    []Update         `json:"ordered_writes,omitempty"`
	ID         string           `json:"operation_id"`
	Hash       string           `json:"contract_sha256"`
	SourceHash string           `json:"source_sha256"`
	NodeID     string           `json:"node_id"`
	Span       frontend.Span    `json:"span"`
	Purpose    string           `json:"purpose_declared"`
	Status     string           `json:"status"`
	Reason     string           `json:"reason,omitempty"`
	Parameters []Parameter      `json:"parameters"`
	Fields     []string         `json:"fields"`
	Reads      []string         `json:"reads"`
	Writes     []string         `json:"writes"`
	Effects    []string         `json:"effects"`
	Post       map[string]*Term `json:"post,omitempty"`
	Return     *Term            `json:"return,omitempty"`
	Checks     []*Term          `json:"checked_steps,omitempty"`
}
type Bundle struct {
	Schema     string     `json:"schema"`
	SourceHash string     `json:"source_sha256"`
	Methods    []Contract `json:"methods"`
}

func Derive(p *frontend.Program) Bundle {
	b := Bundle{Schema: "phlam.contracts.v0", SourceHash: p.SourceHash, Methods: []Contract{}}
	for _, class := range p.Classes {
		for _, m := range class.Children {
			if m.Kind == "method" {
				b.Methods = append(b.Methods, derive(p, class, m))
			}
		}
	}
	return b
}
func derive(p *frontend.Program, class, m *frontend.Node) (c Contract) {
	c = Contract{ID: class.Name + "." + m.Name, Domain: "int64_state_and_arguments", SourceHash: p.SourceHash, NodeID: m.ID, Span: m.Span, Purpose: class.Text,
		Status: "inferred", Parameters: []Parameter{}, Fields: []string{}, Reads: []string{}, Writes: []string{}, Effects: []string{}, Post: map[string]*Term{}, Return: &Term{Op: "null"}}
	defer func() {
		if r := recover(); r != nil {
			if msg, ok := r.(unsupported); ok {
				c.Status = "unknown"
				c.Domain = "unknown"
				c.Reason = string(msg)
				c.Post = nil
				c.Return = nil
				c.Checks = nil
				c.Updates = nil
				c.Reads = nil
				c.Writes = nil
				c.Effects = nil
			} else {
				panic(r)
			}
		}
		raw, _ := json.Marshal(c)
		c.Hash = fmt.Sprintf("%x", sha256.Sum256(raw))
	}()
	sizes := map[*Term]int{}
	var size func(*Term) int
	size = func(t *Term) int {
		if t == nil {
			return 0
		}
		if n, ok := sizes[t]; ok {
			return n
		}
		n := 1 + size(t.Left) + size(t.Right)
		if n > 1024 {
			panic(unsupported("symbolic expression expansion limit exceeded"))
		}
		sizes[t] = n
		return n
	}
	for _, n := range m.Children {
		if n.Kind == "parameter" {
			c.Parameters = append(c.Parameters, Parameter{Name: n.Name, HasDefault: len(n.Children) > 0})
		}
	}
	for _, n := range class.Children {
		if n.Kind == "property" {
			c.Fields = append(c.Fields, n.Name)
		}
	}
	sort.Strings(c.Fields)
	slots := map[string]*Term{}
	locals := map[string]*Term{}
	reads, writes := map[string]bool{}, map[string]bool{}
	for _, n := range class.Children {
		if n.Kind == "property" {
			if len(n.Children) != 1 {
				panic(unsupported("integer contracts require integer property defaults"))
			}
			if _, ok := Constant(n.Children[0]); !ok {
				panic(unsupported("integer contracts require integer property defaults"))
			}
			slots[n.Name] = &Term{Op: "slot", Name: n.Name}
		}
	}
	paramIndex := 0
	for _, n := range m.Children {
		if n.Kind == "parameter" {
			param := c.Parameters[paramIndex]
			if n.Name == "this" {
				panic(unsupported("this cannot be a parameter"))
			}
			locals[n.Name] = &Term{Op: "arg", Name: n.Name}
			if len(n.Children) > 0 {
				v, ok := Constant(n.Children[0])
				if !ok {
					panic(unsupported("parameter default is outside the integer fragment"))
				}
				param.Default = &v
			}
			c.Parameters[paramIndex] = param
			paramIndex++
		}
	}
	var expr func(*frontend.Node) *Term
	expr = func(n *frontend.Node) *Term {
		if v, ok := Constant(n); ok {
			return &Term{Op: "int", Int: v}
		}
		switch n.Kind {
		case "variable":
			if t, ok := locals[n.Name]; ok {
				return t
			}
		case "property_access":
			if n.Children[0].Kind == "variable" && n.Children[0].Name == "this" {
				if t, ok := slots[n.Name]; ok {
					reads[n.Name] = true
					return t
				}
			}
		case "binary":
			op := map[string]string{"+": "add", "-": "sub", "*": "mul"}[n.Text]
			if op != "" {
				t := &Term{Op: op, Left: expr(n.Children[0]), Right: expr(n.Children[1])}
				size(t)
				c.Checks = append(c.Checks, t)
				return t
			}
		case "unary":
			if n.Text == "-" || n.Text == "+" {
				t := expr(n.Children[0])
				if n.Text == "+" {
					return t
				}
				t = &Term{Op: "neg", Left: t}
				size(t)
				c.Checks = append(c.Checks, t)
				return t
			}
		}
		panic(unsupported("expression outside the straight-line integer fragment: " + n.Kind))
	}
	body := m.Children[len(m.Children)-1]
	if len(body.Children) > 256 {
		panic(unsupported("contract statement limit exceeded"))
	}
	for _, s := range body.Children {
		switch s.Kind {
		case "assign":
			lhs, rhs := s.Children[0], s.Children[1]
			var old *Term
			if s.Text != "=" {
				old = expr(lhs)
			}
			term := expr(rhs)
			if s.Text != "=" {
				op := "add"
				if s.Text == "-=" {
					op = "sub"
				}
				term = &Term{Op: op, Left: old, Right: term}
				size(term)
				c.Checks = append(c.Checks, term)
			}
			if lhs.Kind == "variable" && lhs.Name != "this" {
				locals[lhs.Name] = term
			} else if lhs.Kind == "property_access" && lhs.Children[0].Kind == "variable" && lhs.Children[0].Name == "this" && slots[lhs.Name] != nil {
				slots[lhs.Name] = term
				c.Post[lhs.Name] = term
				c.Updates = append(c.Updates, Update{Slot: lhs.Name, Value: term})
				writes[lhs.Name] = true
			} else {
				panic(unsupported("assignment target outside the integer fragment"))
			}
		case "return":
			if len(s.Children) > 0 {
				c.Return = expr(s.Children[0])
			}
			goto complete
		default:
			panic(unsupported("statement outside the straight-line integer fragment: " + s.Kind))
		}
	}
complete:
	for name := range reads {
		c.Reads = append(c.Reads, name)
	}
	sort.Strings(c.Reads)
	for name := range writes {
		c.Writes = append(c.Writes, name)
	}
	sort.Strings(c.Writes)
	if len(c.Writes) > 0 {
		c.Effects = append(c.Effects, "instance_state.write")
	}
	return c
}

type unsupported string

// Constant supports the asymmetric int64 minimum without first overflowing its
// positive token. Other constant expressions are executed as checked steps.
func Constant(n *frontend.Node) (int64, bool) {
	text := ""
	if n.Kind == "literal" {
		text = n.Text
	}
	if n.Kind == "unary" && (n.Text == "-" || n.Text == "+") && n.Children[0].Kind == "literal" {
		text = n.Text + n.Children[0].Text
	}
	v, err := strconv.ParseInt(text, 10, 64)
	return v, err == nil
}

// Validate compares one concrete result/state delta with the derived relation.
// It never mutates its inputs, invokes an AST evaluator, or grants authority.
// The caller must supply normalized effective arguments and complete state maps.
func Validate(c Contract, transition Transition) error {
	before, after, args, result := transition.Before, transition.After, transition.Args, transition.Result
	if c.Status != "inferred" || c.Domain != "int64_state_and_arguments" {
		return fmt.Errorf("unsupported contract: %s", c.Reason)
	}
	if len(args) != len(c.Parameters) {
		return fmt.Errorf("argument frame mismatch")
	}
	for _, p := range c.Parameters {
		v, ok := args[p.Name]
		if !ok || !v.Valid() || v.Kind != "int" {
			return fmt.Errorf("argument %s must be int64", p.Name)
		}
	}
	if len(before) != len(c.Fields) || len(after) != len(c.Fields) {
		return fmt.Errorf("state frame mismatch")
	}
	for _, name := range c.Fields {
		a, ok := before[name]
		b, exists := after[name]
		if !ok || !exists || !a.Valid() || !b.Valid() || a.Kind != "int" || b.Kind != "int" {
			return fmt.Errorf("slot %s must be an existing int64", name)
		}
	}
	var eval func(*Term) (semantics.Value, error)
	eval = func(t *Term) (semantics.Value, error) {
		if t == nil {
			return semantics.Value{}, fmt.Errorf("missing relation term")
		}
		switch t.Op {
		case "null":
			return semantics.Null(), nil
		case "int":
			return semantics.Integer(t.Int), nil
		case "slot":
			v, ok := before[t.Name]
			if !ok {
				return v, fmt.Errorf("unknown slot")
			}
			return v, nil
		case "arg":
			v, ok := args[t.Name]
			if !ok {
				return v, fmt.Errorf("unknown argument")
			}
			return v, nil
		}
		a, err := eval(t.Left)
		if err != nil {
			return a, err
		}
		if a.Kind != "int" {
			return a, fmt.Errorf("integer relation operand required")
		}
		if t.Op == "neg" {
			v, err := semantics.Neg(a.Int)
			return semantics.Integer(v), err
		}
		b, err := eval(t.Right)
		if err != nil {
			return b, err
		}
		if b.Kind != "int" {
			return b, fmt.Errorf("integer relation operand required")
		}
		var v int64
		switch t.Op {
		case "add":
			v, err = semantics.Add(a.Int, b.Int)
		case "sub":
			v, err = semantics.Sub(a.Int, b.Int)
		case "mul":
			v, err = semantics.Mul(a.Int, b.Int)
		default:
			err = fmt.Errorf("unknown relation primitive")
		}
		return semantics.Integer(v), err
	}
	for _, step := range c.Checks {
		if _, err := eval(step); err != nil {
			return fmt.Errorf("checked step: %w", err)
		}
	}
	if len(transition.Writes) != len(c.Updates) {
		return fmt.Errorf("write history length mismatch")
	}
	for i, update := range c.Updates {
		want, err := eval(update.Value)
		if err != nil {
			return err
		}
		got := transition.Writes[i]
		if got.Slot != update.Slot || !got.Value.Valid() || got.Value != want {
			return fmt.Errorf("ordered write relation violated at step %d", i)
		}
	}
	for _, name := range c.Fields {
		want := before[name]
		if term, ok := c.Post[name]; ok {
			v, err := eval(term)
			if err != nil {
				return err
			}
			want = v
		}
		if after[name] != want {
			return fmt.Errorf("state relation violated for slot %s", name)
		}
	}
	want, err := eval(c.Return)
	if err != nil {
		return err
	}
	if !result.Valid() || result != want {
		return fmt.Errorf("return relation violated")
	}
	return nil
}
