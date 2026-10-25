// Package components lowers a restricted scalar class into path relations.
// Guards and effects come from source, never from the purpose comment or model.
package components

import (
	"fmt"
	"strconv"

	"github.com/LynnColeArt/phlam/compiler/frontend"
	abi "github.com/LynnColeArt/phlam/runtime/components"
)

type symbolic struct {
	guard          *abi.Expr
	locals         map[string]*abi.Expr
	post           *abi.Expr
	checks, writes []*abi.Expr
}

func clone(p symbolic) symbolic {
	q := p
	q.locals = map[string]*abi.Expr{}
	for k, v := range p.locals {
		q.locals[k] = v
	}
	q.checks = append([]*abi.Expr(nil), p.checks...)
	q.writes = append([]*abi.Expr(nil), p.writes...)
	return q
}
func Compile(source, className string) (contract abi.Contract, err error) {
	program, err := frontend.Parse(source)
	if err != nil {
		return contract, err
	}
	var class *frontend.Node
	for _, c := range program.Classes {
		if c.Name == className {
			class = c
		}
	}
	if class == nil {
		return contract, fmt.Errorf("class not found")
	}
	contract = abi.Contract{Class: class.Name, Purpose: class.Text, SourceHash: program.SourceHash, Operations: []abi.Operation{}}
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(compileError); ok {
				contract = abi.Contract{}
				err = fmt.Errorf("component contract: %s", e)
			} else {
				panic(r)
			}
		}
	}()
	propertyCount := 0
	var init *frontend.Node
	for _, n := range class.Children {
		if n.Kind == "property" {
			propertyCount++
			contract.Slot = n.Name
			if len(n.Children) != 1 || n.Children[0].Kind != "literal" || n.Children[0].Text != "0" {
				panic(compileError("scalar property must default to zero"))
			}
		}
		if n.Kind == "method" && n.Name == "init" {
			init = n
		}
	}
	if propertyCount != 1 || init == nil || len(init.Children) != 2 || len(init.Children[1].Children) != 1 {
		panic(compileError("require one scalar property and single-argument identity init"))
	}
	stmt := init.Children[1].Children[0]
	if stmt.Kind != "assign" || stmt.Text != "=" || !slot(stmt.Children[0], contract.Slot) || stmt.Children[1].Kind != "variable" || stmt.Children[1].Name != init.Children[0].Name {
		panic(compileError("init must assign its argument to the scalar property"))
	}
	for _, method := range class.Children {
		if method.Kind != "method" || method.Name == "init" {
			continue
		}
		arity := len(method.Children) - 1
		if arity > 1 {
			panic(compileError("at most one integer argument per operation"))
		}
		op := abi.Operation{Name: method.Name, Arity: arity, NodeID: method.ID, Paths: []abi.Path{}}
		initial := symbolic{guard: &abi.Expr{Op: "bool", Bool: true}, post: &abi.Expr{Op: "slot"}, locals: map[string]*abi.Expr{}}
		if arity == 1 {
			if method.Children[0].Name == "this" {
				panic(compileError("this is not a parameter"))
			}
			initial.locals[method.Children[0].Name] = &abi.Expr{Op: "arg"}
		}
		exprDepth := 0
		var expr func(*frontend.Node, *symbolic) *abi.Expr
		expr = func(n *frontend.Node, p *symbolic) *abi.Expr {
			exprDepth++
			defer func() { exprDepth-- }()
			if exprDepth > 32 {
				panic(compileError("expression depth budget exceeded"))
			}
			switch n.Kind {
			case "literal":
				if n.Text == "false" || n.Text == "true" {
					return &abi.Expr{Op: "bool", Bool: n.Text == "true"}
				}
				v, err := strconv.ParseInt(n.Text, 10, 64)
				if err == nil {
					return &abi.Expr{Op: "int", Int: v}
				}
			case "variable":
				if v := p.locals[n.Name]; v != nil {
					return v
				}
			case "property_access":
				if slot(n, contract.Slot) {
					return p.post
				}
			case "unary":
				if n.Text == "-" {
					if n.Children[0].Kind == "literal" {
						if value, err := strconv.ParseInt("-"+n.Children[0].Text, 10, 64); err == nil {
							return &abi.Expr{Op: "int", Int: value}
						}
					}
					v := &abi.Expr{Op: "sub", Left: &abi.Expr{Op: "int"}, Right: expr(n.Children[0], p)}
					p.checks = append(p.checks, v)
					return v
				}
				if n.Text == "!" {
					return &abi.Expr{Op: "not", Left: expr(n.Children[0], p)}
				}
			case "binary":
				name := map[string]string{"+": "add", "-": "sub", "*": "mul", "<": "lt", ">": "gt", "<=": "le", ">=": "ge", "==": "eq", "!=": "ne", "&&": "and", "||": "or"}[n.Text]
				if name != "" {
					v := &abi.Expr{Op: name, Left: expr(n.Children[0], p), Right: expr(n.Children[1], p)}
					if name == "add" || name == "sub" || name == "mul" {
						p.checks = append(p.checks, v)
					}
					return v
				}
			}
			panic(compileError("unsupported expression " + n.Kind))
		}
		var block func([]*frontend.Node, []symbolic) []symbolic
		block = func(statements []*frontend.Node, paths []symbolic) []symbolic {
			for _, s := range statements {
				if len(op.Paths)+len(paths) > 16 {
					panic(compileError("path budget exceeded"))
				}
				next := []symbolic{}
				for _, path := range paths {
					p := clone(path)
					switch s.Kind {
					case "assign":
						lhs := s.Children[0]
						var old *abi.Expr
						if s.Text != "=" {
							old = expr(lhs, &p)
						}
						value := expr(s.Children[1], &p)
						if s.Text != "=" {
							kind := "add"
							if s.Text == "-=" {
								kind = "sub"
							}
							value = &abi.Expr{Op: kind, Left: old, Right: value}
							p.checks = append(p.checks, value)
						}
						if lhs.Kind == "variable" && lhs.Name != "this" {
							checkExpression(value)
							p.locals[lhs.Name] = value
						} else if slot(lhs, contract.Slot) {
							checkExpression(value)
							p.post = value
							p.writes = append(p.writes, value)
						} else {
							panic(compileError("write outside the component frame"))
						}
						if len(p.writes) > 1 {
							panic(compileError("one write per primitive supported"))
						}
						next = append(next, p)
					case "return":
						if len(s.Children) != 1 {
							panic(compileError("exported operations must return int or rejection false"))
						}
						result := expr(s.Children[0], &p)
						checkExpression(p.guard)
						checkExpression(p.post)
						checkExpression(result)
						op.Paths = append(op.Paths, abi.Path{Guard: p.guard, Checks: p.checks, Post: p.post, Writes: p.writes, Return: result})
					case "if":
						condition := expr(s.Children[0], &p)
						yes, no := clone(p), clone(p)
						yes.guard = &abi.Expr{Op: "and", Left: p.guard, Right: condition}
						no.guard = &abi.Expr{Op: "and", Left: p.guard, Right: &abi.Expr{Op: "not", Left: condition}}
						next = append(next, block(s.Children[1].Children, []symbolic{yes})...)
						if len(s.Children) > 2 {
							next = append(next, block(s.Children[2].Children, []symbolic{no})...)
						} else {
							next = append(next, no)
						}
					default:
						panic(compileError("unsupported statement " + s.Kind))
					}
				}
				paths = next
			}
			return paths
		}
		remaining := block(method.Children[arity].Children, []symbolic{initial})
		if len(remaining) > 0 {
			panic(compileError("all paths must return"))
		}
		contract.Operations = append(contract.Operations, op)
	}
	if len(contract.Operations) == 0 {
		panic(compileError("no exported operations"))
	}
	return contract, nil
}

type compileError string

func slot(n *frontend.Node, name string) bool {
	return n.Kind == "property_access" && n.Name == name && len(n.Children) == 1 && n.Children[0].Kind == "variable" && n.Children[0].Name == "this"
}

func checkExpression(e *abi.Expr) {
	nodes := 0
	var visit func(*abi.Expr, int)
	visit = func(e *abi.Expr, depth int) {
		if e == nil {
			return
		}
		nodes++
		if nodes > 512 || depth > 32 {
			panic(compileError("symbolic expression budget exceeded"))
		}
		visit(e.Left, depth+1)
		visit(e.Right, depth+1)
	}
	visit(e, 0)
}
