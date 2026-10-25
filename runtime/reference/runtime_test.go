package reference

import (
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/LynnColeArt/phlam/compiler/frontend"
	"github.com/LynnColeArt/phlam/semantics"
)

func counterSource(t *testing.T) string {
	t.Helper()
	s, err := os.ReadFile("../../examples/counter/counter.phlam")
	if err != nil {
		t.Fatal(err)
	}
	return string(s)
}
func runtime(t *testing.T, source string, options Options) *Runtime {
	t.Helper()
	r, err := New(source, options)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func construct(t *testing.T, r *Runtime, class string, args ...int64) semantics.Value {
	t.Helper()
	values := []semantics.Value{}
	for _, n := range args {
		values = append(values, semantics.Integer(n))
	}
	v, err := r.Construct(class, values...)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func call(t *testing.T, r *Runtime, h semantics.Value, method string, args ...int64) semantics.Value {
	t.Helper()
	values := []semantics.Value{}
	for _, n := range args {
		values = append(values, semantics.Integer(n))
	}
	v, err := r.Call(h, method, values...)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func wantError(t *testing.T, err error, code string) {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || e.Code != code || e.NodeID == "" || e.Span.Line < 1 {
		t.Fatalf("want %s with provenance, got %v", code, err)
	}
}
func TestCounterIdentityDefaultsAndVersions(t *testing.T) {
	r := runtime(t, counterSource(t), Options{})
	a := construct(t, r, "Counter", 10)
	alias := a
	b := construct(t, r, "Counter")
	if a == b {
		t.Fatal("new reused identity")
	}
	if got := call(t, r, alias, "increment", 2); got != semantics.Integer(12) {
		t.Fatal(got)
	}
	if call(t, r, a, "read") != semantics.Integer(12) || call(t, r, b, "read") != semantics.Integer(0) {
		t.Fatal("identity or independence broken")
	}
	if call(t, r, a, "increment") != semantics.Integer(13) {
		t.Fatal("default delta broken")
	}
	objects := r.Objects()
	if objects[0].Version != 3 || objects[1].Version != 1 {
		t.Fatal("read changed version or mutation missing", objects)
	}
	objects[0].Fields["value"] = semantics.Integer(99)
	if call(t, r, a, "read").Int != 13 {
		t.Fatal("snapshot exposed mutable heap")
	}
	bundle := r.Contracts()
	bundle.Methods[1].Post["value"].Op = "null"
	if call(t, r, a, "increment", -2).Int != 11 {
		t.Fatal("exported contract modified runtime")
	}
	_, err := r.Call(a, "init", semantics.Integer(1))
	wantError(t, err, "R_INIT")
	_, err = r.Call(a, "increment", semantics.Boolean(true))
	wantError(t, err, "R_TYPE")
	_, err = r.Call(a, "read", semantics.Integer(1))
	wantError(t, err, "R_ARITY")
}
func TestSourceAssignmentAndNestedCalls(t *testing.T) {
	source := counterSource(t) + "\n/// Exercises object identity in source.\nclass Demo {function run() {$a=new Counter(10); $b=$a; $b->increment(2); $other=new Counter(); return $a->read()-$other->read()}}"
	r := runtime(t, source, Options{CaptureIntegers: true})
	demo := construct(t, r, "Demo")
	if call(t, r, demo, "run").Int != 12 {
		t.Fatal("source assignment did not preserve object identity")
	}
	found := false
	for _, e := range r.Events() {
		if e.Parent != 0 && e.Operation == "call_enter" {
			found = true
		}
	}
	if !found {
		t.Fatal("nested call causality missing")
	}
}
func TestOverflowRollbackAndFailureTrace(t *testing.T) {
	r := runtime(t, counterSource(t), Options{CaptureIntegers: true})
	h := construct(t, r, "Counter", math.MaxInt64)
	_, err := r.Call(h, "increment")
	wantError(t, err, "R_OVERFLOW")
	if call(t, r, h, "read").Int != math.MaxInt64 || r.Objects()[0].Version != 1 {
		t.Fatal("overflow changed state/version")
	}
	failedTx := uint64(2)
	failed, rollback := false, false
	for _, e := range r.Events() {
		if e.Transaction == failedTx {
			if e.Phase == "committed" {
				t.Fatal("failed transaction committed")
			}
			failed = failed || e.Phase == "failed"
			rollback = rollback || e.Phase == "rolled_back"
		}
	}
	if !failed || !rollback {
		t.Fatal("failure evidence missing")
	}
	// A prior nested call writes successfully, then the outer method fails. All
	// writes, versions, and new objects in the outer transaction must roll back.
	source := counterSource(t) + "\n/// Fails after a nested state write.\nclass Driver {function fail($target) {$target->increment(1); $fresh=new Counter(3); return 9223372036854775807+1}}"
	r = runtime(t, source, Options{})
	target := construct(t, r, "Counter", 1)
	driver := construct(t, r, "Driver")
	before, _ := r.SnapshotJSON()
	_, err = r.Call(driver, "fail", target)
	wantError(t, err, "R_OVERFLOW")
	after, _ := r.SnapshotJSON()
	if string(before) != string(after) {
		t.Fatal("outer rollback leaked nested state or allocation")
	}
}
func TestConstructionArityLiteralsAndBudgets(t *testing.T) {
	source := "/// State.\nclass State {$value=0; function init($start) {$this->value=$start; $bad=9223372036854775807+1}}"
	r := runtime(t, source, Options{})
	_, err := r.Construct("State", semantics.Integer(3))
	wantError(t, err, "R_OVERFLOW")
	if len(r.Objects()) != 0 {
		t.Fatal("failed init published instance")
	}
	r = runtime(t, "/// Minimum.\nclass Minimum {$value=-9223372036854775808; function read(){return $this->value}}", Options{})
	h := construct(t, r, "Minimum")
	if call(t, r, h, "read").Int != math.MinInt64 {
		t.Fatal("minimum literal corrupted")
	}
	r = runtime(t, "/// Requires init.\nclass Required {function init($start){}}", Options{})
	_, err = r.Construct("Required")
	wantError(t, err, "R_ARITY")
	r = runtime(t, counterSource(t), Options{MaxSteps: 1})
	_, err = r.Construct("Counter")
	wantError(t, err, "R_BUDGET")
	if len(r.Objects()) != 0 {
		t.Fatal("budget failure published instance")
	}
	r = runtime(t, "/// Recursion.\nclass Recur {function run(){return $this->run()}}", Options{MaxDepth: 3})
	h = construct(t, r, "Recur")
	_, err = r.Call(h, "run")
	wantError(t, err, "R_DEPTH")
	var e *Error
	errors.As(err, &e)
	if len(e.Stack) != 3 {
		t.Fatal("causal stack missing")
	}
}
func TestDeterministicTracesAndOptInCapture(t *testing.T) {
	source := counterSource(t)
	execute := func(capture bool) ([]byte, *Runtime) {
		r := runtime(t, source, Options{CaptureIntegers: capture})
		h := construct(t, r, "Counter", 4)
		call(t, r, h, "increment", 2)
		data, err := json.Marshal(r.Events())
		if err != nil {
			t.Fatal(err)
		}
		return data, r
	}
	a, r := execute(true)
	b, _ := execute(true)
	if string(a) != string(b) {
		t.Fatal("traces are nondeterministic")
	}
	p, err := frontend.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	nodes := map[string]bool{}
	var walk func(*frontend.Node)
	walk = func(n *frontend.Node) {
		nodes[n.ID] = true
		for _, c := range n.Children {
			walk(c)
		}
	}
	for _, c := range p.Classes {
		walk(c)
	}
	sawWrite, sawAdd, sawCheck := false, false, false
	for i, e := range r.Events() {
		if e.Sequence != i || e.Transaction == 0 || !nodes[e.NodeID] || e.Span.Line < 1 {
			t.Fatal("invalid trace provenance", e)
		}
		sawWrite = sawWrite || e.Operation == "stage_write"
		sawAdd = sawAdd || e.Operation == "integer_add"
		sawCheck = sawCheck || e.Operation == "contract_validated"
	}
	if !sawWrite || !sawAdd || !sawCheck {
		t.Fatal("missing actual execution observations")
	}
	events := r.Events()
	for i := range events {
		for key := range events[i].Integers {
			events[i].Integers[key] = 99
		}
	}
	unchanged, _ := json.Marshal(r.Events())
	if string(a) != string(unchanged) {
		t.Fatal("trace copy exposed mutable data")
	}
	metadata, _ := execute(false)
	if strings.Contains(string(metadata), "integers") {
		t.Fatal("values captured without opt-in")
	}
}

func TestStrictConditionsAndUnsupportedFeatures(t *testing.T) {
	source := "/// Tests a selected branch.\nclass Guard {$value=0; function choose($x) {if ($x > 0) {return 7} else {return 8}} function invalid() {if (1) {return 7}} function skip() {return false && (9223372036854775807+1 > 0)} function dynamic() {$name='value'; $$name=1} function eval_case(){return eval('return 1')}}"
	r := runtime(t, source, Options{})
	h := construct(t, r, "Guard")
	if call(t, r, h, "choose", 1).Int != 7 || call(t, r, h, "choose", 0).Int != 8 || call(t, r, h, "skip") != semantics.Boolean(false) {
		t.Fatal("branch or short circuit wrong")
	}
	_, err := r.Call(h, "invalid")
	wantError(t, err, "R_TYPE")
	_, err = r.Call(h, "dynamic")
	wantError(t, err, "R_LITERAL")
	_, err = r.Call(h, "eval_case")
	wantError(t, err, "R_UNSUPPORTED")
	_, err = r.Call(semantics.Handle(999), "read")
	wantError(t, err, "R_HANDLE")
	_, err = r.Call(h, "missing")
	wantError(t, err, "R_NAME")
}

func FuzzCounterTransition(f *testing.F) {
	data, err := os.ReadFile("../../examples/counter/counter.phlam")
	if err != nil {
		f.Fatal(err)
	}
	for _, pair := range [][2]int64{{0, 1}, {10, 2}, {math.MaxInt64, 1}, {math.MinInt64, -1}, {math.MinInt64, math.MaxInt64}, {math.MaxInt64, -1}} {
		f.Add(pair[0], pair[1])
	}
	f.Fuzz(func(t *testing.T, start, delta int64) {
		r := runtime(t, string(data), Options{})
		h := construct(t, r, "Counter", start)
		got, err := r.Call(h, "increment", semantics.Integer(delta))
		expected := new(big.Int).Add(big.NewInt(start), big.NewInt(delta))
		if expected.IsInt64() {
			if err != nil || got != semantics.Integer(expected.Int64()) {
				t.Fatalf("wrong transition %d + %d: %v %v", start, delta, got, err)
			}
		} else {
			wantError(t, err, "R_OVERFLOW")
			if r.Objects()[0].Fields["value"].Int != start || r.Objects()[0].Version != 1 {
				t.Fatal("overflow published state")
			}
		}
	})
}
