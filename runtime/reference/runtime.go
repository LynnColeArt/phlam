// Package reference implements the small exact oracle, not the deployed learned
// executor. Each public New/Call is a transaction on a private heap snapshot.
package reference

import (
	"encoding/json"
	"sort"

	"github.com/LynnColeArt/phlam/compiler/contracts"
	"github.com/LynnColeArt/phlam/compiler/frontend"
	"github.com/LynnColeArt/phlam/runtime/trace"
	"github.com/LynnColeArt/phlam/semantics"
)

type Options struct {
	CaptureIntegers bool
	MaxSteps        int
	MaxDepth        int
}
type Object struct {
	ID      uint64                     `json:"id"`
	Class   string                     `json:"class"`
	Version uint64                     `json:"version"`
	Fields  map[string]semantics.Value `json:"fields"`
}
type Frame struct {
	Invocation uint64        `json:"invocation"`
	Operation  string        `json:"operation"`
	NodeID     string        `json:"node_id"`
	Span       frontend.Span `json:"span"`
}
type Error struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	NodeID  string        `json:"node_id"`
	Span    frontend.Span `json:"span"`
	Stack   []Frame       `json:"stack"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

type writeRecord struct {
	object uint64
	write  contracts.Write
}
type Runtime struct {
	writes           []writeRecord
	transactionID    uint64
	root             Frame
	program          *frontend.Program
	classes          map[string]*frontend.Node
	methods          map[string]map[string]*frontend.Node
	contracts        map[string]contracts.Contract
	heap             map[uint64]*Object
	nextID, nextCall uint64
	options          Options
	recorder         trace.Recorder
	working          map[uint64]*Object
	dirty            map[uint64]*frontend.Node
	frames           []Frame
	steps            int
}

func New(source string, options Options) (*Runtime, error) {
	p, err := frontend.Parse(source)
	if err != nil {
		return nil, err
	}
	if options.MaxSteps <= 0 {
		options.MaxSteps = 10000
	}
	if options.MaxDepth <= 0 {
		options.MaxDepth = 64
	}
	r := &Runtime{program: p, classes: map[string]*frontend.Node{}, methods: map[string]map[string]*frontend.Node{}, contracts: map[string]contracts.Contract{}, heap: map[uint64]*Object{}, options: options}
	for _, c := range p.Classes {
		r.classes[c.Name] = c
		r.methods[c.Name] = map[string]*frontend.Node{}
		for _, m := range c.Children {
			if m.Kind == "method" {
				r.methods[c.Name][m.Name] = m
			}
		}
	}
	for _, c := range contracts.Derive(p).Methods {
		r.contracts[c.ID] = c
	}
	return r, nil
}
func (r *Runtime) SourceHash() string { return r.program.SourceHash }
func (r *Runtime) Contracts() contracts.Bundle {
	// Derive afresh so caller mutations cannot alter the runtime's checked relations.
	return contracts.Derive(r.program)
}
func (r *Runtime) Events() []trace.Event { return r.recorder.Events() }
func cloneFields(m map[string]semantics.Value) map[string]semantics.Value {
	out := make(map[string]semantics.Value, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
func cloneHeap(m map[uint64]*Object) map[uint64]*Object {
	out := make(map[uint64]*Object, len(m))
	for id, o := range m {
		copy := *o
		copy.Fields = cloneFields(o.Fields)
		out[id] = &copy
	}
	return out
}
func (r *Runtime) Objects() []Object {
	ids := make([]uint64, 0, len(r.heap))
	for id := range r.heap {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	out := make([]Object, 0, len(ids))
	for _, id := range ids {
		o := *r.heap[id]
		o.Fields = cloneFields(o.Fields)
		out = append(out, o)
	}
	return out
}
func (r *Runtime) fail(n *frontend.Node, code, msg string) {
	panic(&Error{Code: code, Message: msg, NodeID: n.ID, Span: n.Span, Stack: append([]Frame(nil), r.frames...)})
}
func (r *Runtime) tick(n *frontend.Node) {
	r.steps--
	if r.steps < 0 {
		r.fail(n, "R_BUDGET", "execution step budget exhausted")
	}
}
func (r *Runtime) event(n *frontend.Node, phase, op string, id uint64, ints map[string]int64, code string) {
	e := trace.Event{Transaction: r.transactionID, Phase: phase, NodeID: n.ID, Span: n.Span, Object: id, Operation: op, Code: code}
	if len(r.frames) == 0 {
		e.Invocation = r.root.Invocation
		e.ContractID = r.root.Operation
	}
	if len(r.frames) > 0 {
		f := r.frames[len(r.frames)-1]
		e.Invocation = f.Invocation
		e.ContractID = f.Operation
		if len(r.frames) > 1 {
			e.Parent = r.frames[len(r.frames)-2].Invocation
		}
	}
	if o := r.working[id]; o != nil {
		e.StateVersion = o.Version
	}
	if r.options.CaptureIntegers {
		e.Integers = ints
	}
	r.recorder.Record(e)
}
func (r *Runtime) transaction(n *frontend.Node, fn func() semantics.Value) (result semantics.Value, err error) {
	r.transactionID++
	r.root = Frame{}
	r.working = cloneHeap(r.heap)
	r.dirty = map[uint64]*frontend.Node{}
	r.writes = nil
	r.steps = r.options.MaxSteps
	defer func() {
		if recovered := recover(); recovered != nil {
			if e, ok := recovered.(*Error); ok {
				err = e
				result = semantics.Null()
				r.recorder.Record(trace.Event{Transaction: r.transactionID, Phase: "failed", Operation: "invocation_failed", Code: e.Code, NodeID: e.NodeID, Span: e.Span, Invocation: lastInvocation(e.Stack)})
				r.event(n, "rolled_back", "transaction_rollback", 0, nil, e.Code)
			} else {
				panic(recovered)
			}
		}
		r.frames = nil
		r.working = nil
		r.dirty = nil
	}()
	r.event(n, "observed", "transaction_begin", 0, nil, "")
	result = fn()
	ids := make([]uint64, 0, len(r.dirty))
	for id := range r.dirty {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		r.working[id].Version++
		r.event(r.dirty[id], "committed", "state_commit", id, nil, "")
	}
	r.heap = r.working
	r.event(n, "committed", "transaction_commit", 0, nil, "")
	return result, nil
}
func lastInvocation(frames []Frame) uint64 {
	if len(frames) == 0 {
		return 0
	}
	return frames[len(frames)-1].Invocation
}
func (r *Runtime) Construct(class string, args ...semantics.Value) (semantics.Value, error) {
	n := r.classes[class]
	if n == nil {
		site := r.program.Classes[0]
		return r.transaction(site, func() semantics.Value { r.fail(site, "R_NAME", "unknown class"); return semantics.Null() })
	}
	return r.transaction(n, func() semantics.Value { return r.construct(n, args) })
}
func (r *Runtime) Call(target semantics.Value, method string, args ...semantics.Value) (semantics.Value, error) {
	n := r.program.Classes[0]
	return r.transaction(n, func() semantics.Value { return r.call(n, target, method, args, false) })
}
func (r *Runtime) object(n *frontend.Node, v semantics.Value) *Object {
	if !v.Valid() || v.Kind != "object" {
		r.fail(n, "R_TYPE", "object handle required")
	}
	o := r.working[v.Object]
	if o == nil {
		r.fail(n, "R_HANDLE", "unknown object handle")
	}
	return o
}
func (r *Runtime) construct(n *frontend.Node, args []semantics.Value) semantics.Value {
	class := r.classes[n.Name]
	if class == nil {
		r.fail(n, "R_NAME", "unknown class")
	}
	r.tick(n)
	r.nextID++
	id := r.nextID
	o := &Object{ID: id, Class: class.Name, Fields: map[string]semantics.Value{}}
	r.working[id] = o
	r.dirty[id] = n
	r.event(n, "observed", "allocate", id, nil, "")
	for _, prop := range class.Children {
		if prop.Kind == "property" {
			r.tick(prop)
			v := semantics.Null()
			if len(prop.Children) > 0 {
				var ok bool
				var number int64
				number, ok = contracts.Constant(prop.Children[0])
				if !ok {
					r.fail(prop, "R_UNSUPPORTED", "property default must be an int64 literal in this slice")
				}
				v = semantics.Integer(number)
			}
			o.Fields[prop.Name] = v
			r.event(prop, "observed", "property_default", id, intValue(v), "")
		}
	}
	if r.methods[class.Name]["init"] != nil {
		r.call(n, semantics.Handle(id), "init", args, true)
	} else if len(args) > 0 {
		r.fail(n, "R_ARITY", "class without init accepts no arguments")
	}
	return semantics.Handle(id)
}
func intValue(v semantics.Value) map[string]int64 {
	if v.Kind == "int" {
		return map[string]int64{"value": v.Int}
	}
	return nil
}
func (r *Runtime) call(site *frontend.Node, target semantics.Value, name string, args []semantics.Value, constructing bool) semantics.Value {
	r.tick(site)
	o := r.object(site, target)
	if name == "init" && !constructing {
		r.fail(site, "R_INIT", "init is callable only during construction")
	}
	method := r.methods[o.Class][name]
	if method == nil {
		r.fail(site, "R_NAME", "unknown method")
	}
	if len(r.frames) >= r.options.MaxDepth {
		r.fail(site, "R_DEPTH", "call depth budget exhausted")
	}
	r.nextCall++
	frame := Frame{Invocation: r.nextCall, Operation: o.Class + "." + name, NodeID: method.ID, Span: method.Span}
	if len(r.frames) == 0 {
		r.root = frame
	}
	r.frames = append(r.frames, frame)
	defer func() { r.frames = r.frames[:len(r.frames)-1] }()
	r.event(method, "observed", "call_enter", o.ID, nil, "")
	c := r.contracts[frame.Operation]
	locals := map[string]semantics.Value{"this": target}
	parameters := map[string]semantics.Value{}
	count := len(method.Children) - 1
	if len(args) > count {
		r.fail(site, "R_ARITY", "too many arguments")
	}
	for i, p := range method.Children[:count] {
		if p.Name == "this" {
			r.fail(p, "R_THIS", "this cannot be a parameter")
		}
		var v semantics.Value
		if i < len(args) {
			v = args[i]
		} else if len(p.Children) > 0 {
			v = r.eval(p.Children[0], map[string]semantics.Value{})
		} else {
			r.fail(site, "R_ARITY", "missing required argument")
		}
		if c.Status == "inferred" && v.Kind != "int" {
			r.fail(p, "R_TYPE", "integer contract argument required")
		}
		if !v.Valid() {
			r.fail(site, "R_TYPE", "invalid value representation")
		}
		locals[p.Name] = v
		parameters[p.Name] = v
		r.event(p, "observed", "bind_argument", o.ID, intValue(v), "")
	}
	before := cloneFields(o.Fields)
	writeStart := len(r.writes)
	result, returned := r.block(method.Children[count], locals)
	if !returned {
		result = semantics.Null()
	}
	if c.Status == "inferred" {
		writes := make([]contracts.Write, 0, len(r.writes)-writeStart)
		for _, record := range r.writes[writeStart:] {
			if record.object != o.ID {
				r.fail(method, "R_CONTRACT", "write outside the instance frame")
			}
			writes = append(writes, record.write)
		}
		if err := contracts.Validate(c, contracts.Transition{Before: before, After: o.Fields, Args: parameters, Result: result, Writes: writes}); err != nil {
			r.fail(method, "R_CONTRACT", err.Error())
		}
		r.event(method, "observed", "contract_validated", o.ID, nil, "")
	} else {
		r.event(method, "observed", "contract_unknown", o.ID, nil, "UNKNOWN_CONTRACT")
	}
	r.event(method, "observed", "call_return", o.ID, intValue(result), "")
	return result
}
func (r *Runtime) block(n *frontend.Node, locals map[string]semantics.Value) (semantics.Value, bool) {
	for _, s := range n.Children {
		r.tick(s)
		switch s.Kind {
		case "return":
			v := semantics.Null()
			if len(s.Children) > 0 {
				v = r.eval(s.Children[0], locals)
			}
			r.event(s, "observed", "return", 0, intValue(v), "")
			return v, true
		case "expression_statement":
			r.eval(s.Children[0], locals)
		case "assign":
			lhs := s.Children[0]
			var object *Object
			switch lhs.Kind {
			case "variable":
				if lhs.Name == "this" {
					r.fail(lhs, "R_THIS", "this binding is read-only")
				}
			case "property_access":
				object = r.object(lhs, r.eval(lhs.Children[0], locals))
				if _, ok := object.Fields[lhs.Name]; !ok {
					r.fail(lhs, "R_NAME", "undefined property")
				}
			default:
				r.fail(lhs, "R_UNSUPPORTED", "assignment target unsupported in this slice")
			}
			old := semantics.Null()
			if s.Text != "=" {
				if object != nil {
					old = object.Fields[lhs.Name]
					r.event(lhs, "observed", "state_read", object.ID, intValue(old), "")
				} else {
					var ok bool
					old, ok = locals[lhs.Name]
					if !ok {
						r.fail(lhs, "R_NAME", "undefined variable")
					}
				}
			}
			rhs := r.eval(s.Children[1], locals)
			if s.Text != "=" {
				op := "+"
				if s.Text == "-=" {
					op = "-"
				}
				rhs = r.arithmetic(s, op, old, rhs)
			}
			if object != nil {
				object.Fields[lhs.Name] = rhs
				r.writes = append(r.writes, writeRecord{object: object.ID, write: contracts.Write{Slot: lhs.Name, Value: rhs}})
				r.dirty[object.ID] = s
				r.event(s, "observed", "stage_write", object.ID, intValue(rhs), "")
			} else {
				locals[lhs.Name] = rhs
				r.event(s, "observed", "local_write", 0, intValue(rhs), "")
			}
		case "if":
			v := r.eval(s.Children[0], locals)
			if v.Kind != "bool" {
				r.fail(s, "R_TYPE", "condition must be boolean")
			}
			branch := 1
			decision := int64(1)
			if !v.Bool {
				branch = 2
				decision = 0
			}
			r.event(s, "observed", "branch", 0, map[string]int64{"taken": decision}, "")
			if branch < len(s.Children) {
				if result, returned := r.block(s.Children[branch], locals); returned {
					return result, true
				}
			}
		default:
			r.fail(s, "R_UNSUPPORTED", "unsupported statement: "+s.Kind)
		}
	}
	return semantics.Null(), false
}
func (r *Runtime) arithmetic(n *frontend.Node, op string, a, b semantics.Value) semantics.Value {
	if a.Kind != "int" || b.Kind != "int" {
		r.fail(n, "R_TYPE", "arithmetic requires int64 operands")
	}
	var v int64
	var err error
	switch op {
	case "+":
		v, err = semantics.Add(a.Int, b.Int)
	case "-":
		v, err = semantics.Sub(a.Int, b.Int)
	case "*":
		v, err = semantics.Mul(a.Int, b.Int)
	default:
		r.fail(n, "R_UNSUPPORTED", "unsupported integer operator")
	}
	if err != nil {
		r.fail(n, "R_OVERFLOW", err.Error())
	}
	r.event(n, "observed", "integer_"+map[string]string{"+": "add", "-": "sub", "*": "mul"}[op], 0, map[string]int64{"left": a.Int, "right": b.Int, "result": v}, "")
	return semantics.Integer(v)
}
func (r *Runtime) eval(n *frontend.Node, locals map[string]semantics.Value) semantics.Value {
	r.tick(n)
	if v, ok := contracts.Constant(n); ok {
		value := semantics.Integer(v)
		r.event(n, "observed", "literal", 0, intValue(value), "")
		return value
	}
	switch n.Kind {
	case "literal":
		switch n.Text {
		case "true":
			return semantics.Boolean(true)
		case "false":
			return semantics.Boolean(false)
		case "null":
			return semantics.Null()
		}
		r.fail(n, "R_LITERAL", "only int64, boolean and null literals execute in this slice")
	case "variable":
		v, ok := locals[n.Name]
		if !ok {
			r.fail(n, "R_NAME", "undefined variable")
		}
		r.event(n, "observed", "local_read", 0, intValue(v), "")
		return v
	case "property_access":
		o := r.object(n, r.eval(n.Children[0], locals))
		v, ok := o.Fields[n.Name]
		if !ok {
			r.fail(n, "R_NAME", "undefined property")
		}
		r.event(n, "observed", "state_read", o.ID, intValue(v), "")
		return v
	case "new":
		args := r.arguments(n.Children, locals)
		return r.construct(n, args)
	case "call":
		callee := n.Children[0]
		if callee.Kind != "property_access" {
			r.fail(n, "R_UNSUPPORTED", "only instance method calls execute in this slice")
		}
		target := r.eval(callee.Children[0], locals)
		args := r.arguments(n.Children[1:], locals)
		return r.call(n, target, callee.Name, args, false)
	case "binary":
		a := r.eval(n.Children[0], locals)
		if n.Text == "&&" || n.Text == "||" {
			if a.Kind != "bool" {
				r.fail(n, "R_TYPE", "boolean operand required")
			}
			if n.Text == "&&" && !a.Bool || n.Text == "||" && a.Bool {
				return a
			}
			b := r.eval(n.Children[1], locals)
			if b.Kind != "bool" {
				r.fail(n, "R_TYPE", "boolean operand required")
			}
			return b
		}
		b := r.eval(n.Children[1], locals)
		if n.Text == "+" || n.Text == "-" || n.Text == "*" {
			return r.arithmetic(n, n.Text, a, b)
		}
		if n.Text == "==" {
			return semantics.Boolean(a == b)
		}
		if n.Text == "!=" {
			return semantics.Boolean(a != b)
		}
		if a.Kind != "int" || b.Kind != "int" {
			r.fail(n, "R_TYPE", "comparison requires int64 operands")
		}
		switch n.Text {
		case "<":
			return semantics.Boolean(a.Int < b.Int)
		case ">":
			return semantics.Boolean(a.Int > b.Int)
		case "<=":
			return semantics.Boolean(a.Int <= b.Int)
		case ">=":
			return semantics.Boolean(a.Int >= b.Int)
		}
	case "unary":
		a := r.eval(n.Children[0], locals)
		if n.Text == "!" {
			if a.Kind != "bool" {
				r.fail(n, "R_TYPE", "boolean operand required")
			}
			return semantics.Boolean(!a.Bool)
		}
		if n.Text == "+" {
			if a.Kind != "int" {
				r.fail(n, "R_TYPE", "integer operand required")
			}
			return a
		}
		if n.Text == "-" {
			return r.arithmetic(n, "-", semantics.Integer(0), a)
		}
	}
	r.fail(n, "R_UNSUPPORTED", "unsupported expression: "+n.Kind)
	return semantics.Null()
}
func (r *Runtime) arguments(nodes []*frontend.Node, locals map[string]semantics.Value) []semantics.Value {
	out := make([]semantics.Value, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, r.eval(n, locals))
	}
	return out
}

// SnapshotJSON is a local debugging aid, never a production observation endpoint.
func (r *Runtime) SnapshotJSON() ([]byte, error) { return json.Marshal(r.Objects()) }

// Invocation is one method call in an atomic local composition. It is an oracle
// harness operation, not a new top-level language construct or agent authority.
type Invocation struct {
	Method string
	Args   []semantics.Value
}

func (r *Runtime) Compose(target semantics.Value, calls []Invocation) (results []semantics.Value, err error) {
	site := r.program.Classes[0]
	_, err = r.transaction(site, func() semantics.Value {
		for _, call := range calls {
			results = append(results, r.call(site, target, call.Method, call.Args, false))
		}
		return semantics.Null()
	})
	if err != nil {
		results = nil
	}
	return results, err
}
