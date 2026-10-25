package contracts

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/LynnColeArt/phlam/compiler/frontend"
	"github.com/LynnColeArt/phlam/semantics"
)

func deriveSource(t *testing.T, source string) Bundle {
	t.Helper()
	p, err := frontend.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	return Derive(p)
}
func counter(t *testing.T) Bundle {
	t.Helper()
	s, err := os.ReadFile("../../examples/counter/counter.phlam")
	if err != nil {
		t.Fatal(err)
	}
	return deriveSource(t, string(s))
}
func state(n int64) map[string]semantics.Value {
	return map[string]semantics.Value{"value": semantics.Integer(n)}
}
func TestCounterRelationsAndFrames(t *testing.T) {
	b := counter(t)
	if len(b.Methods) != 3 {
		t.Fatal("missing contracts")
	}
	c := b.Methods[1]
	if c.ID != "Counter.increment" || c.Status != "inferred" || !reflect.DeepEqual(c.Writes, []string{"value"}) || c.Post["value"].Op != "add" {
		t.Fatalf("wrong contract %+v", c)
	}
	valid := func() Transition {
		return Transition{Before: state(10), After: state(12), Args: map[string]semantics.Value{"delta": semantics.Integer(2)}, Result: semantics.Integer(12), Writes: []Write{{"value", semantics.Integer(12)}}}
	}
	if err := Validate(c, valid()); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		change func(*Transition)
	}{
		{"wrong delta", func(x *Transition) {
			x.After = state(13)
			x.Result = semantics.Integer(13)
			x.Writes[0].Value = semantics.Integer(13)
		}},
		{"wrong return", func(x *Transition) { x.Result = semantics.Integer(11) }},
		{"extra slot", func(x *Transition) { x.After["secret"] = semantics.Integer(1) }},
		{"missing slot", func(x *Transition) { delete(x.After, "value") }},
		{"bad argument", func(x *Transition) { x.Args["delta"] = semantics.Boolean(true) }},
		{"missing argument", func(x *Transition) { x.Args = nil }},
		{"extra argument", func(x *Transition) { x.Args["other"] = semantics.Integer(1) }},
		{"hidden write", func(x *Transition) { x.Writes = append(x.Writes, Write{"value", semantics.Integer(12)}) }},
		{"wrong intermediate write", func(x *Transition) { x.Writes[0].Value = semantics.Integer(11) }},
		{"overflow", func(x *Transition) {
			x.Before = state(math.MaxInt64)
			x.After = state(math.MinInt64)
			x.Args["delta"] = semantics.Integer(1)
			x.Result = semantics.Integer(math.MinInt64)
			x.Writes[0].Value = x.Result
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			x := valid()
			test.change(&x)
			before, _ := json.Marshal(x)
			if Validate(c, x) == nil {
				t.Fatal("invalid transition accepted")
			}
			after, _ := json.Marshal(x)
			if string(before) != string(after) {
				t.Fatal("checker mutated proposal")
			}
		})
	}
	read := b.Methods[2]
	if err := Validate(read, Transition{Before: state(12), After: state(12), Args: map[string]semantics.Value{}, Result: semantics.Integer(12), Writes: []Write{{"value", semantics.Integer(12)}}}); err == nil {
		t.Fatal("read accepted a same-value write")
	}
	data, _ := json.Marshal(b)
	data2, _ := json.Marshal(counter(t))
	if string(data) != string(data2) {
		t.Fatal("nondeterministic contracts")
	}
}
func TestFrameOrderAndIntermediateOverflow(t *testing.T) {
	b := deriveSource(t, "/// Claims read-only, but actually writes.\nclass Pair {$value=0; $guard=0; function run($delta) {$this->value=1; $this->value=$delta; return $this->value}}")
	c := b.Methods[0]
	x := Transition{Before: map[string]semantics.Value{"value": semantics.Integer(0), "guard": semantics.Integer(7)}, After: map[string]semantics.Value{"value": semantics.Integer(2), "guard": semantics.Integer(7)}, Args: map[string]semantics.Value{"delta": semantics.Integer(2)}, Result: semantics.Integer(2), Writes: []Write{{"value", semantics.Integer(1)}, {"value", semantics.Integer(2)}}}
	if err := Validate(c, x); err != nil {
		t.Fatal(err)
	}
	x.Writes[0], x.Writes[1] = x.Writes[1], x.Writes[0]
	if Validate(c, x) == nil {
		t.Fatal("reordered writes accepted")
	}
	x.Writes[0], x.Writes[1] = x.Writes[1], x.Writes[0]
	x.After["guard"] = semantics.Integer(8)
	if Validate(c, x) == nil {
		t.Fatal("frame mutation accepted")
	}
	if len(c.Effects) != 1 || c.Effects[0] != "instance_state.write" {
		t.Fatal("purpose prose overrode inferred effect")
	}
	c = deriveSource(t, "/// Overflow.\nclass Counter {$value=0; function run() {$unused=$this->value+1; return $this->value}}").Methods[0]
	if Validate(c, Transition{Before: state(math.MaxInt64), After: state(math.MaxInt64), Result: semantics.Integer(math.MaxInt64)}) == nil {
		t.Fatal("intermediate overflow lost during substitution")
	}
}
func TestUnknownAndBoundedDerivation(t *testing.T) {
	for _, source := range []string{
		"/// Branch.\nclass Branch {$value=0; function run() {if (true) {return 1} else {return 0}}}",
		"/// Null field.\nclass NullField {$value; function run() {return $this->value}}",
		"/// Calls.\nclass Calls {function run() {return eval('return 1')}}",
		"/// Expanded.\nclass Expanded {$value=0; function run() {$x=$this->value;" + strings.Repeat("$x=$x+$x;", 20) + "return $x}}",
	} {
		c := deriveSource(t, source).Methods[0]
		if c.Status != "unknown" || c.Reason == "" || c.Post != nil || c.Updates != nil || c.Hash == "" {
			t.Fatalf("unsupported contract misrepresented %+v", c)
		}
		if Validate(c, Transition{}) == nil {
			t.Fatal("unknown contract accepted")
		}
	}
}
