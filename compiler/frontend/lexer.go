package frontend

import (
	"strings"
	"unicode/utf8"
)

type token struct {
	kind, text               string
	start, end, line, column int
}

func problem(t token, code, msg, hint string) *Diagnostic {
	return &Diagnostic{code, msg, hint, Span{t.start, t.end, t.line, t.column}}
}
func identStart(c byte) bool { return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func ident(c byte) bool      { return identStart(c) || c >= '0' && c <= '9' }

func lex(s string) ([]token, error) {
	var out []token
	i, line, col := 0, 1, 1
	advance := func(end int) {
		for i < end {
			if s[i] == '\n' {
				line++
				col = 1
			} else {
				col++
			}
			i++
		}
	}
	emit := func(kind string, end int) {
		t := token{kind, s[i:end], i, end, line, col}
		advance(end)
		out = append(out, t)
	}
	for i < len(s) {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\r' {
			advance(i + 1)
			continue
		}
		if c == '\n' {
			emit("nl", i+1)
			continue
		}
		if strings.HasPrefix(s[i:], "//") || c == '#' {
			end := i
			for end < len(s) && s[end] != '\n' {
				end++
			}
			if strings.HasPrefix(s[i:], "///") {
				emit("doc", end)
			} else {
				advance(end)
			}
			continue
		}
		if strings.HasPrefix(s[i:], "/*") {
			t := token{start: i, end: i + 2, line: line, column: col}
			n := strings.Index(s[i+2:], "*/")
			if n < 0 {
				return nil, problem(t, "P001", "unterminated block comment", "close the comment with */")
			}
			end := i + 2 + n + 2
			// Ordinary multiline comments preserve statement boundaries.
			for i < end {
				if s[i] == '\n' {
					emit("nl", i+1)
				} else {
					advance(i + 1)
				}
			}
			continue
		}
		if identStart(c) {
			end := i + 1
			for end < len(s) && ident(s[end]) {
				end++
			}
			emit("ident", end)
			continue
		}
		if c >= '0' && c <= '9' {
			end := i + 1
			for end < len(s) && s[end] >= '0' && s[end] <= '9' {
				end++
			}
			emit("int", end)
			continue
		}
		if c == '\'' || c == '"' {
			end := i + 1
			for end < len(s) && s[end] != c && s[end] != '\n' && s[end] != '\r' {
				if s[end] == '\\' {
					end++
					if end >= len(s) || s[end] == '\n' || s[end] == '\r' {
						break
					}
				}
				end++
			}
			if end >= len(s) || s[end] != c {
				return nil, problem(token{start: i, end: i + 1, line: line, column: col}, "P002", "unterminated string", "close the string on the same line")
			}
			emit("string", end+1)
			continue
		}
		matched := false
		for _, op := range []string{"->", "+=", "-=", "==", "!=", "<=", ">=", "&&", "||", "=>"} {
			if strings.HasPrefix(s[i:], op) {
				emit(op, i+len(op))
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		if strings.ContainsRune("{}()[]$;,+-*/%=!<>&.", rune(c)) {
			emit(string(c), i+1)
			continue
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		return nil, problem(token{start: i, end: i + size, line: line, column: col}, "P003", "unsupported character", "use the ASCII bootstrap syntax; PHP opening tags are not supported")
	}
	out = append(out, token{"eof", "", i, i, line, col})
	return out, nil
}
