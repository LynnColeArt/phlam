package frontend

import (
	"fmt"
	"strings"
)

type parser struct {
	tokens                   []token
	i, depth, layout, blocks int
	hash                     string
}

// Parse does syntax and local declaration validation, not effect analysis or
// name resolution. Unsupported language features fail instead of being ignored.
func Parse(source string) (program *Program, err error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens, hash: hashSource(source)}
	// Recursive descent uses a private typed panic to unwind the first diagnostic.
	defer func() {
		if r := recover(); r != nil {
			if d, ok := r.(*Diagnostic); ok {
				program = nil
				err = d
			} else {
				panic(r)
			}
		}
	}()
	program = &Program{Schema: "phlam.ast.v0", SourceHash: p.hash, Classes: []*Node{}}
	names := map[string]bool{}
	p.lines()
	for !p.at("eof") {
		var purpose []string
		for p.at("doc") {
			purpose = append(purpose, strings.TrimSpace(strings.TrimPrefix(p.take().text, "///")))
			p.lines()
		}
		if !p.at("class") {
			p.fail("P100", "only documented classes are allowed at top level", "move executable statements and functions into a class method")
		}
		if strings.TrimSpace(strings.Join(purpose, "\n")) == "" {
			p.fail("P101", "class requires a nonempty purpose comment", "put /// followed by the class purpose immediately before class")
		}
		n := p.class(strings.Join(purpose, "\n"))
		if names[n.Name] {
			p.fail("P102", "duplicate class "+n.Name, "give each class a unique name")
		}
		names[n.Name] = true
		program.Classes = append(program.Classes, n)
		p.lines()
	}
	if len(program.Classes) == 0 {
		p.fail("P100", "source contains no classes", "declare at least one documented class")
	}
	return program, nil
}
func (p *parser) current() token { return p.tokens[p.i] }
func (p *parser) at(k string) bool {
	t := p.current()
	return t.kind == k || t.kind == "ident" && t.text == k
}
func (p *parser) take() token {
	t := p.current()
	if t.kind != "eof" {
		p.i++
	}
	return t
}
func (p *parser) accept(k string) bool {
	if p.at(k) {
		p.take()
		return true
	}
	return false
}
func (p *parser) fail(code, msg, hint string) { panic(problem(p.current(), code, msg, hint)) }
func (p *parser) need(k string) token {
	if !p.at(k) {
		p.fail("P104", fmt.Sprintf("expected %s, found %q", k, p.current().text), "check the declaration or expression syntax")
	}
	return p.take()
}
func (p *parser) lines() {
	for p.accept("nl") {
	}
}
func (p *parser) name() token {
	t := p.need("ident")
	switch t.text {
	case "class", "function", "return", "if", "else", "new", "true", "false", "null", "for", "while", "var", "public", "private", "protected", "eval":
		panic(problem(t, "P105", "reserved word used as a name", "choose an ordinary identifier"))
	}
	return t
}
func pascal(s string) bool { return s != "" && s[0] >= 'A' && s[0] <= 'Z' && !strings.Contains(s, "_") }
func snake(s string) bool {
	for i, c := range s {
		if !(c >= 'a' && c <= 'z' || c == '_' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return s != ""
}
func (p *parser) class(purpose string) *Node {
	t := p.need("class")
	name := p.name().text
	if !pascal(name) {
		p.fail("P106", "class name must use PascalCase", "start the class name with an uppercase letter and omit underscores")
	}
	p.lines()
	p.need("{")
	p.lines()
	var members []*Node
	seen := map[string]bool{}
	for !p.at("}") && !p.at("eof") {
		var n *Node
		if p.at("$") {
			start := p.take()
			n = p.node("property", start)
			n.Name = p.name().text
			if p.accept("=") {
				n.Children = []*Node{p.expr(0)}
			}
			p.endStatement()
			n.Span.End = p.tokens[p.i-1].end
		} else if p.at("function") {
			n = p.method(name)
		} else {
			p.fail("P107", "expected property or method", "use $property or function method_name(...); omit var and visibility modifiers")
		}
		key := n.Kind + ":" + n.Name
		if seen[key] {
			p.fail("P108", "duplicate member "+n.Name, "remove the duplicate declaration")
		}
		seen[key] = true
		members = append(members, n)
		p.lines()
	}
	p.need("}")
	n := p.node("class", t, members...)
	n.Name = name
	n.Text = purpose
	return n
}
func (p *parser) method(class string) *Node {
	t := p.need("function")
	name := p.name().text
	if strings.EqualFold(name, class) || strings.HasPrefix(name, "__") {
		p.fail("P109", "unsupported constructor or magic method", "use function init(...) as the sole initializer")
	}
	if !snake(name) {
		p.fail("P110", "method name must use snake_case", "use lowercase words separated by underscores")
	}
	p.need("(")
	p.lines()
	var children []*Node
	seen := map[string]bool{}
	defaulted := false
	for !p.at(")") {
		start := p.need("$")
		param := p.node("parameter", start)
		param.Name = p.name().text
		if seen[param.Name] {
			p.fail("P111", "duplicate parameter", "use unique parameter names")
		}
		seen[param.Name] = true
		if p.accept("=") {
			p.lines()
			p.depth++
			param.Children = []*Node{p.expr(0)}
			p.depth--
			defaulted = true
		} else if defaulted {
			p.fail("P112", "required parameter follows a default", "put required parameters before optional ones")
		}
		param.Span.End = p.tokens[p.i-1].end
		children = append(children, param)
		p.lines()
		if !p.accept(",") {
			break
		}
		p.lines()
	}
	p.need(")")
	p.lines()
	children = append(children, p.block())
	n := p.node("method", t, children...)
	n.Name = name
	return n
}
func (p *parser) block() *Node {
	p.blocks++
	defer func() { p.blocks-- }()
	if p.blocks > 128 {
		p.fail("P116", "block nesting limit exceeded", "reduce block nesting below 128 levels")
	}
	t := p.need("{")
	p.lines()
	var children []*Node
	for !p.at("}") && !p.at("eof") {
		children = append(children, p.statement())
		p.lines()
	}
	p.need("}")
	return p.node("block", t, children...)
}
func (p *parser) statement() *Node {
	t := p.current()
	if p.accept("return") {
		var children []*Node
		if !p.at("nl") && !p.at(";") && !p.at("}") {
			children = append(children, p.expr(0))
		}
		p.endStatement()
		return p.node("return", t, children...)
	}
	if p.accept("if") {
		p.need("(")
		p.layout++
		p.depth++
		p.lines()
		condition := p.expr(0)
		p.lines()
		p.depth--
		p.layout--
		p.need(")")
		p.lines()
		children := []*Node{condition, p.block()}
		p.lines()
		if p.accept("else") {
			p.lines()
			children = append(children, p.block())
		}
		return p.node("if", t, children...)
	}
	lhs := p.expr(0)
	if p.at("=") || p.at("+=") || p.at("-=") {
		if !assignable(lhs) {
			p.fail("P113", "assignment requires a writable location", "assign to a variable, property, or array item")
		}
		op := p.take()
		p.lines()
		rhs := p.expr(0)
		p.endStatement()
		n := p.node("assign", t, lhs, rhs)
		n.Text = op.text
		return n
	}
	p.endStatement()
	return p.node("expression_statement", t, lhs)
}
func assignable(n *Node) bool {
	return n.Kind == "variable" || n.Kind == "dynamic_variable" || n.Kind == "property_access" || n.Kind == "index"
}
func (p *parser) endStatement() {
	if p.accept(";") {
		p.lines()
		return
	}
	if p.at("}") || p.at("eof") {
		return
	}
	if p.accept("nl") {
		p.lines()
		switch p.current().kind {
		case "+", "-", "*", "/", "%", ".", "(", "[", "->", "==", "!=", "<", ">", "<=", ">=", "&&", "||", "=", "+=", "-=":
			p.fail("P114", "ambiguous continuation after a completed statement", "wrap a multiline expression in parentheses, or use an explicit semicolon")
		}
		return
	}
	p.fail("P115", "expected a statement boundary", "separate statements with a newline or semicolon")
}

var precedence = map[string]int{"||": 1, "&&": 2, "==": 3, "!=": 3, "<": 4, ">": 4, "<=": 4, ">=": 4, "+": 5, "-": 5, ".": 5, "*": 6, "/": 6, "%": 6}

func (p *parser) expr(min int) *Node {
	// Bound recursive parsing before Go's growing stack can be exhausted.
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 256 {
		p.fail("P116", "expression nesting limit exceeded", "reduce nesting below 256 levels")
	}
	t := p.current()
	var left *Node
	switch {
	case p.at("$"):
		left = p.variable()
	case p.at("int") || p.at("string") || p.at("true") || p.at("false") || p.at("null"):
		p.take()
		left = p.node("literal", t)
		left.Text = t.text
	case p.accept("new"):
		name := p.name().text
		if !pascal(name) {
			p.fail("P117", "new requires a class name", "use new PascalCase(...)")
		}
		args := p.arguments()
		left = p.node("new", t, args...)
		left.Name = name
	case p.at("-") || p.at("+") || p.at("!") || p.at("&"):
		op := p.take()
		p.lines()
		child := p.expr(7)
		if op.kind == "&" && !assignable(child) {
			p.fail("P118", "alias requires a writable location", "use &$variable or &$object->property")
		}
		left = p.node("unary", t, child)
		left.Text = op.text
	case p.accept("("):
		p.layout++
		p.lines()
		left = p.expr(0)
		p.lines()
		p.need(")")
		p.layout--
	case p.accept("["):
		p.layout++
		p.lines()
		var items []*Node
		for !p.at("]") {
			start := p.current()
			item := p.expr(0)
			if p.accept("=>") {
				p.lines()
				item = p.node("entry", start, item, p.expr(0))
			}
			items = append(items, item)
			p.lines()
			if !p.accept(",") {
				break
			}
			p.lines()
		}
		p.need("]")
		left = p.node("array", t, items...)
		p.layout--
	case p.at("ident"):
		if p.at("eval") {
			p.take()
			left = p.node("name", t)
			left.Name = "eval"
		} else {
			left = p.node("name", t)
			left.Name = p.name().text
		}
	default:
		p.fail("P119", "expected an expression", "use a literal, variable, call, array, or new ClassName(...)")
	}
	for {
		if p.layout > 0 {
			p.lines()
		}
		// depth includes recursive expression nesting; parentheses/arrays/calls
		// explicitly consume layout around delimiters. A completed expression never
		// consumes a top-level newline here.
		if p.at("->") && 8 >= min {
			p.take()
			name := p.name()
			left = p.node("property_access", t, left)
			left.Name = name.text
			continue
		}
		if p.at("[") && 8 >= min {
			p.take()
			p.layout++
			p.lines()
			idx := p.expr(0)
			p.lines()
			p.need("]")
			p.layout--
			left = p.node("index", t, left, idx)
			continue
		}
		if p.at("(") && 8 >= min {
			if left.Kind == "property_access" && left.Name == "init" {
				p.fail("P120", "init cannot be called as an ordinary method", "construct a fresh instance with new ClassName(...)")
			}
			args := p.arguments()
			left = p.node("call", t, append([]*Node{left}, args...)...)
			continue
		}
		prec := precedence[p.current().kind]
		if prec == 0 || prec < min {
			break
		}
		op := p.take()
		p.lines()
		rhs := p.expr(prec + 1)
		left = p.node("binary", t, left, rhs)
		left.Text = op.text
	}
	return left
}
func (p *parser) arguments() []*Node {
	p.layout++
	defer func() { p.layout-- }()
	p.need("(")
	p.lines()
	var args []*Node
	for !p.at(")") {
		args = append(args, p.expr(0))
		p.lines()
		if !p.accept(",") {
			break
		}
		p.lines()
	}
	p.need(")")
	return args
}

func (p *parser) variable() *Node {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 256 {
		p.fail("P116", "variable nesting limit exceeded", "reduce indirect variable nesting below 256 levels")
	}
	t := p.need("$")
	if p.at("$") {
		return p.node("dynamic_variable", t, p.variable())
	}
	name := p.name().text
	n := p.node("variable", t)
	n.Name = name
	return n
}
