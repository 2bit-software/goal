// Package main is a standalone reference transpiler for goal feature 03-result:
// `Result[T, error]` as the whole-return error channel, in the open-E common case.
// It implements the §8.3 keystone: a `Result[T, error]` consumed immediately lowers
// to Go's native (T, error). Construction `Result.Ok(v)` -> (v, nil), `Result.Err(e)`
// -> (zero, e); a `match` on the Result lowers to the idiomatic `if err != nil` /
// `else` the model already knows.
//
// Scope: open-E only (E is `error`). Closed-E (sum encoding) is feature 06; `?`
// propagation is feature 05. This transpiler does NO error checking — no must-use
// tracking, no exhaustiveness, no type inference. It handles the IMMEDIATE case
// (§8.7): a Result that is returned and match-ed at the use site. A Result stored as
// a first-class value (slice/map/field) must fall back to the sum encoding and is
// out of scope here. Malformed input is undefined behavior.
//
// Positions handled: function return type, `return Result.Ok/Err(...)`, and
// statement-position `match call { Result.Ok(b) => ...; Result.Err(b) => ... }`.
// Value-position Result match and stored Results are deferred (see classifyStmt).
package main

import (
	"fmt"
	"go/format"
	"sort"
	"strings"
	"text/scanner"
)

// token is a lexical token with its byte span in the source.
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

// replacement is a byte span of the source to splice over with generated Go.
type replacement struct {
	start, end int
	text       string
}

// hygienic temporaries (§8 prefix).
const (
	okName  = "__goal_ok"  // named success return / Ok binding target
	errName = "__goal_err" // named error return / Err binding target
	valName = "__goal_v"   // Ok value captured at a match site
)

// transpile lowers goal source using Result[T, error] into idiomatic Go.
func transpile(src string) (string, error) {
	toks := lex(src)
	var reps []replacement

	// Pass 1: rewrite `func ... Result[T, error] {` to named returns `(__goal_ok T,
	// __goal_err error)`. Named returns make the Err-path zero value (`__goal_ok`)
	// available without synthesizing a type-specific zero literal.
	for i := range toks {
		if toks[i].text == "func" {
			if rep, ok := rewriteSignature(src, toks, i); ok {
				reps = append(reps, rep)
			}
		}
	}

	// Pass 2: lower `return Result.Ok(X)` / `return Result.Err(X)`.
	for i := 0; i+4 < len(toks); i++ {
		if toks[i].text == "return" && toks[i+1].text == "Result" && toks[i+2].text == "." &&
			(toks[i+3].text == "Ok" || toks[i+3].text == "Err") && toks[i+4].text == "(" {
			close := matchParen(toks, i+4)
			inner := strings.TrimSpace(src[toks[i+4].end:toks[close].start])
			var text string
			if toks[i+3].text == "Ok" {
				text = inner + ", nil"
			} else {
				text = okName + ", " + inner
			}
			reps = append(reps, replacement{toks[i+1].start, toks[close].end, text})
			i = close
		}
	}

	// Pass 3: lower statement-position `match call { Result.Ok/Err arms }`.
	for i := 0; i < len(toks); i++ {
		if toks[i].text == "match" {
			rep, next, err := lowerResultMatch(src, toks, i)
			if err != nil {
				return "", err
			}
			reps = append(reps, rep)
			i = next
		}
	}

	out := splice(src, 0, len(src), reps)
	formatted, err := format.Source([]byte(out))
	if err != nil {
		return "", fmt.Errorf("generated Go did not parse: %w\n--- generated ---\n%s", err, out)
	}
	return string(formatted), nil
}

// ----- pass 1: signature rewrite -----

// rewriteSignature rewrites the return type of a `func ... Result[T, error]` to
// named Go returns. It returns false (no replacement) for functions that do not
// return a Result.
func rewriteSignature(src string, toks []token, fi int) (replacement, bool) {
	body := firstBodyBrace(toks, fi)
	if body < 0 {
		return replacement{}, false
	}
	pc := paramsClose(toks, body)
	if pc < 0 || pc+2 >= body {
		return replacement{}, false
	}
	if toks[pc+1].text != "Result" || toks[pc+2].text != "[" {
		return replacement{}, false
	}
	rb := matchBracket(toks, pc+2)
	comma := topLevelComma(toks, pc+2, rb)
	if comma < 0 {
		return replacement{}, false
	}
	t := strings.TrimSpace(src[toks[pc+3].start:toks[comma].start])
	text := fmt.Sprintf("(%s %s, %s error)", okName, t, errName)
	return replacement{toks[pc+1].start, toks[rb].end, text}, true
}

// firstBodyBrace returns the index of the function body's opening "{": the first
// "{" at paren/bracket depth 0 after the `func` at fi.
func firstBodyBrace(toks []token, fi int) int {
	depth := 0
	for k := fi + 1; k < len(toks); k++ {
		switch toks[k].text {
		case "(", "[":
			depth++
		case ")", "]":
			depth--
		case "{":
			if depth == 0 {
				return k
			}
		}
	}
	return -1
}

// paramsClose returns the index of the ")" closing the parameter list: the first
// ")" at depth 0 scanning back from the body brace (skipping a balanced return type
// such as Result[T, error], and never reaching the receiver's parens).
func paramsClose(toks []token, body int) int {
	depth := 0
	for k := body - 1; k >= 0; k-- {
		t := toks[k].text
		if depth == 0 && t == ")" {
			return k
		}
		switch t {
		case ")", "]":
			depth++
		case "(", "[":
			depth--
		}
	}
	return -1
}

// ----- pass 3: Result match -----

type resultArm struct {
	variant string // "Ok" or "Err"
	binding string // "" if none
	bodyLo  int
	bodyHi  int
}

func lowerResultMatch(src string, toks []token, mi int) (replacement, int, error) {
	if mi > 0 {
		if p := toks[mi-1].text; p == "return" || p == "=" {
			return replacement{}, 0, fmt.Errorf("value-position Result match is deferred in the reference transpiler; consume a Result with a statement-position match (open-E keystone, §8.3)")
		}
	}

	bo := mi + 1
	for depth := 0; ; bo++ {
		switch toks[bo].text {
		case "(", "[":
			depth++
		case ")", "]":
			depth--
		case "{":
			if depth == 0 {
				goto found
			}
		}
	}
found:
	scrut := strings.TrimSpace(src[toks[mi].end:toks[bo].start])
	bc := matchBrace(toks, bo)
	arms := parseResultArms(toks, bo+1, bc)

	var ok, errArm *resultArm
	for i := range arms {
		switch arms[i].variant {
		case "Ok":
			ok = &arms[i]
		case "Err":
			errArm = &arms[i]
		}
	}
	if ok == nil || errArm == nil {
		return replacement{}, 0, fmt.Errorf("Result match must have both Result.Ok and Result.Err arms")
	}

	okBody, okUsed := rewriteResultBody(src, toks, *ok, valName)
	errBody, _ := rewriteResultBody(src, toks, *errArm, errName)

	okLHS := "_"
	if okUsed {
		okLHS = valName
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s, %s := %s\n", okLHS, errName, scrut)
	fmt.Fprintf(&b, "if %s != nil {\n%s\n} else {\n%s\n}", errName, errBody, okBody)
	return replacement{toks[mi].start, toks[bc].end, b.String()}, bc + 1, nil
}

// parseResultArms splits arm-block tokens [lo, hi) into arms, delimited by `=>`
// (lexed as `=` `>`). The pattern is `Result.Ok(b)` / `Result.Err(b)` (or without a
// binding); its tokens sit just before each arrow.
func parseResultArms(toks []token, lo, hi int) []resultArm {
	var arrows []int
	for j, depth := lo, 0; j < hi; j++ {
		switch toks[j].text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		}
		if depth == 0 && toks[j].text == "=" && j+1 < hi && toks[j+1].text == ">" {
			arrows = append(arrows, j)
		}
	}

	arms := make([]resultArm, len(arrows))
	for i, eq := range arrows {
		patStart := lo
		if i > 0 {
			patStart = patternStart(toks, arrows[i])
		}
		a := parseResultPattern(toks, patStart, eq)
		a.bodyLo = eq + 2
		if i+1 < len(arrows) {
			a.bodyHi = patternStart(toks, arrows[i+1])
		} else {
			a.bodyHi = hi
		}
		arms[i] = a
	}
	return arms
}

// patternStart finds where the pattern ending just before the arrow at eqIdx begins.
func patternStart(toks []token, eqIdx int) int {
	j := eqIdx - 1
	if toks[j].text == ")" {
		// Result . Variant ( binding ) — walk back to "(" then to "Result".
		depth := 0
		k := j
		for ; k >= 0; k-- {
			switch toks[k].text {
			case ")":
				depth++
			case "(":
				depth--
			}
			if depth == 0 {
				break
			}
		}
		return k - 3
	}
	// Result . Variant
	return j - 2
}

func parseResultPattern(toks []token, start, eqIdx int) resultArm {
	a := resultArm{variant: toks[start+2].text}
	if start+3 < eqIdx && toks[start+3].text == "(" {
		a.binding = toks[start+4].text
	}
	return a
}

// rewriteResultBody renames the arm's payload binding (the whole Ok value or the
// error) to target throughout the body. The bool reports whether the binding was
// referenced (so the caller knows whether to capture the value or discard with `_`).
func rewriteResultBody(src string, toks []token, a resultArm, target string) (string, bool) {
	lo, hi := a.bodyLo, a.bodyHi
	if lo >= hi {
		return "", false
	}
	used := false
	var reps []replacement
	if a.binding != "" {
		for j := lo; j < hi; j++ {
			if toks[j].text == a.binding {
				used = true
				reps = append(reps, replacement{toks[j].start, toks[j].end, target})
			}
		}
	}
	return splice(src, toks[lo].start, toks[hi-1].end, reps), used
}

// ----- shared helpers -----

func matchParen(toks []token, openIdx int) int { return matchPair(toks, openIdx, "(", ")") }
func matchBracket(toks []token, openIdx int) int {
	return matchPair(toks, openIdx, "[", "]")
}

// matchBrace returns the index of the "}" matching the "{" at openIdx.
func matchBrace(toks []token, openIdx int) int { return matchPair(toks, openIdx, "{", "}") }

func matchPair(toks []token, openIdx int, open, close string) int {
	depth := 0
	for k := openIdx; k < len(toks); k++ {
		switch toks[k].text {
		case open:
			depth++
		case close:
			depth--
		}
		if depth == 0 {
			return k
		}
	}
	return len(toks) - 1
}

// topLevelComma returns the index of the "," at the top level between the brackets
// openIdx ("[") and closeIdx ("]").
func topLevelComma(toks []token, openIdx, closeIdx int) int {
	depth := 0
	for k := openIdx + 1; k < closeIdx; k++ {
		switch toks[k].text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		}
		if depth == 0 && toks[k].text == "," {
			return k
		}
	}
	return -1
}

// splice rebuilds src[lo:hi] with each replacement span swapped for its text.
func splice(src string, lo, hi int, reps []replacement) string {
	sort.Slice(reps, func(a, b int) bool { return reps[a].start < reps[b].start })
	var b strings.Builder
	prev := lo
	for _, r := range reps {
		if r.start < prev {
			continue // defensive: skip overlap
		}
		b.WriteString(src[prev:r.start])
		b.WriteString(r.text)
		prev = r.end
	}
	b.WriteString(src[prev:hi])
	return b.String()
}
