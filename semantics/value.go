// Package semantics defines the exact values and checked integer primitives
// shared by the reference oracle and the transition-contract checker.
package semantics

import (
	"encoding/json"
	"errors"
	"math"
)

type Value struct {
	Kind   string `json:"kind"`
	Int    int64  `json:"int,omitempty"`
	Bool   bool   `json:"bool,omitempty"`
	Object uint64 `json:"object,omitempty"`
}

func Integer(n int64) Value  { return Value{Kind: "int", Int: n} }
func Boolean(b bool) Value   { return Value{Kind: "bool", Bool: b} }
func Null() Value            { return Value{Kind: "null"} }
func Handle(id uint64) Value { return Value{Kind: "object", Object: id} }
func (v Value) Valid() bool {
	switch v.Kind {
	case "int":
		return !v.Bool && v.Object == 0
	case "bool":
		return v.Int == 0 && v.Object == 0
	case "null":
		return v.Int == 0 && !v.Bool && v.Object == 0
	case "object":
		return v.Int == 0 && !v.Bool && v.Object != 0
	}
	return false
}

var ErrOverflow = errors.New("signed int64 overflow")

func Add(a, b int64) (int64, error) {
	if b > 0 && a > math.MaxInt64-b || b < 0 && a < math.MinInt64-b {
		return 0, ErrOverflow
	}
	return a + b, nil
}
func Sub(a, b int64) (int64, error) {
	if b < 0 && a > math.MaxInt64+b || b > 0 && a < math.MinInt64+b {
		return 0, ErrOverflow
	}
	return a - b, nil
}
func Neg(a int64) (int64, error) {
	if a == math.MinInt64 {
		return 0, ErrOverflow
	}
	return -a, nil
}
func Mul(a, b int64) (int64, error) {
	if a == 0 || b == 0 {
		return 0, nil
	}
	if a == math.MinInt64 && b == -1 || b == math.MinInt64 && a == -1 {
		return 0, ErrOverflow
	}
	r := a * b
	if r/b != a {
		return 0, ErrOverflow
	}
	return r, nil
}

// MarshalJSON preserves explicit zero/false payloads without emitting irrelevant
// fields. Invalid representations are never exported as valid typed values.
func (v Value) MarshalJSON() ([]byte, error) {
	if !v.Valid() {
		return nil, errors.New("invalid exact value")
	}
	switch v.Kind {
	case "int":
		return json.Marshal(struct {
			Kind string `json:"kind"`
			Int  int64  `json:"int"`
		}{v.Kind, v.Int})
	case "bool":
		return json.Marshal(struct {
			Kind string `json:"kind"`
			Bool bool   `json:"bool"`
		}{v.Kind, v.Bool})
	case "object":
		return json.Marshal(struct {
			Kind   string `json:"kind"`
			Object uint64 `json:"object"`
		}{v.Kind, v.Object})
	default:
		return []byte(`{"kind":"null"}`), nil
	}
}
