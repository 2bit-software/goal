// Package main is a standalone reference transpiler for goal feature 10-assert:
// the runtime assertion statement `assert <cond> [, <fmt> [, <args>...]]` (§4.3).
// Asserts are checked at runtime and are RUNTIME-PRESERVED (§8.6): unlike the
// erased static guarantees, the check must survive into the generated Go.
//
//	assert amount > 0
//	  ->  if !(amount > 0) { panic("assertion failed: amount > 0") }
//
//	assert age >= 0, "got %d", age
//	  ->  if !(age >= 0) {
//	          panic("assertion failed: age >= 0: " + fmt.Sprintf("got %d", age))
//	      }
//
// The panic message ALWAYS includes the source expression text (§8.6's
// located-feedback rule for runtime failures). The optional printf-style message
// is appended via fmt.Sprintf; bare asserts need no message argument and no fmt
// import. The expression text is emitted as a quoted string literal (never as a
// format string), so a `%` in the condition is harmless.
//
// Scope: the statically-checkable assert subset is RESERVED, not built (§4.3); the
// build-tag strip toggle (§8.6) is documented but not implemented in v1 (asserts
// are always emitted). Conditions are single-line and statement-positioned (the
// `assert` keyword is the first token on its line). Malformed input is UB.
package main

import (
	"fmt"
	"go/format"
	"sort"
	"strconv"
	"strings"
	"text/scanner"
)

type token struct {
	text  string
	start int
	end   int
}

func lex(src string) []token {
	var s scanner.Scanner
	s.Init(strings.NewReader(src))
	s.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanRawStrings |
		scanner.ScanInts | scanner.ScanFloats | scanner.ScanChars | scanner.ScanComments | scanner.SkipComments
	s.Whitespace = 1<<'\t' | 1<<'\n' | 1<<'\r' | 1<<' '
	var toks []token
	for tk := s.Scan(); tk != scanner.EOF; tk = s.Scan() {
		txt := s.TokenText()
		start := s.Position.Offset
		toks = append(toks, token{text: txt, start: start, end: start + len(txt)})
	}
	return toks
}

type replacement struct {
	start, end int
	text       string
}

// transpile lowers each `assert` statement to a runtime `if !(cond) { panic(...) }`
// and passes everything else through verbatim.
func transpile(src string) (string, error) {
	toks := lex(src)

	var reps []replacement
	needsFmt := false
	for i := range toks {
		if toks[i].text != "assert" || !isLineStart(src, toks[i].start) {
			continue
		}
		lineEnd := nextNewline(src, toks[i].end)

		// Split the statement at the first top-level comma: left is the condition,
		// right (if any) is the printf-style message `"fmt", args...`.
		commaStart := firstTopLevelComma(toks, i+1, lineEnd)
		var cond, msg string
		if commaStart < 0 {
			cond = trimStmt(src[toks[i].end:lineEnd])
		} else {
			cond = trimStmt(src[toks[i].end:commaStart])
			msg = trimStmt(src[commaStart+1 : lineEnd])
		}

		var block string
		if msg == "" {
			block = fmt.Sprintf("if !(%s) { panic(%s) }",
				cond, strconv.Quote("assertion failed: "+cond))
		} else {
			needsFmt = true
			block = fmt.Sprintf("if !(%s) { panic(%s + fmt.Sprintf(%s)) }",
				cond, strconv.Quote("assertion failed: "+cond+": "), msg)
		}
		reps = append(reps, replacement{toks[i].start, lineEnd, block})
	}

	if needsFmt && !importsFmt(toks) {
		if pos := packageLineEnd(src, toks); pos >= 0 {
			reps = append(reps, replacement{pos, pos, "\n\nimport \"fmt\""})
		}
	}

	out := splice(src, reps)
	formatted, err := format.Source([]byte(out))
	if err != nil {
		return "", fmt.Errorf("generated Go did not parse: %w\n--- generated ---\n%s", err, out)
	}
	return string(formatted), nil
}

// isLineStart reports whether everything between the previous newline and p is
// whitespace — i.e. the token at p begins a statement (so `assert` is the keyword,
// not an identifier used mid-expression).
func isLineStart(src string, p int) bool {
	for k := p - 1; k >= 0; k-- {
		switch src[k] {
		case '\n':
			return true
		case ' ', '\t':
			continue
		default:
			return false
		}
	}
	return true
}

// nextNewline returns the offset of the next '\n' at or after p, or len(src).
func nextNewline(src string, p int) int {
	if nl := strings.IndexByte(src[p:], '\n'); nl >= 0 {
		return p + nl
	}
	return len(src)
}

// firstTopLevelComma returns the byte offset of the first comma between token
// index `from` and byte offset `lineEnd` that sits at bracket depth 0 (so commas
// inside a call like `clamp(lo, hi)` are skipped). Returns -1 if none.
func firstTopLevelComma(toks []token, from, lineEnd int) int {
	depth := 0
	for k := from; k < len(toks) && toks[k].start < lineEnd; k++ {
		switch toks[k].text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		case ",":
			if depth == 0 {
				return toks[k].start
			}
		}
	}
	return -1
}

// trimStmt trims surrounding whitespace and a trailing statement-ending `;`.
func trimStmt(s string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), ";"))
}

// importsFmt reports whether the source already imports the fmt package.
func importsFmt(toks []token) bool {
	for i := range toks {
		if toks[i].text == `"fmt"` {
			return true
		}
	}
	return false
}

// packageLineEnd returns the offset of the newline ending the `package <name>`
// clause, where an injected import can be inserted. Returns -1 if not found.
func packageLineEnd(src string, toks []token) int {
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].text == "package" {
			return nextNewline(src, toks[i+1].end)
		}
	}
	return -1
}

func splice(src string, reps []replacement) string {
	sort.Slice(reps, func(a, b int) bool { return reps[a].start < reps[b].start })
	var b strings.Builder
	prev := 0
	for _, r := range reps {
		if r.start < prev {
			continue
		}
		b.WriteString(src[prev:r.start])
		b.WriteString(r.text)
		prev = r.end
	}
	b.WriteString(src[prev:])
	return b.String()
}
