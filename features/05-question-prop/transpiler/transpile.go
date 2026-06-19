// Package main is a standalone reference transpiler for goal feature
// 05-question-prop: postfix `?` propagation over Result and Option (spec §3.7, §8.3).
// `?` early-returns the Err/None from the enclosing function and unwraps the
// Ok/Some otherwise. It composes the open-E Result lowering (feature 03) and the
// Option pointer strategy (feature 04), so this transpiler also lowers the
// Result/Option signatures and constructions those `?` sites depend on.
//
// `?` is always the RHS of an assignment: `name := expr?` keeps the unwrapped value;
// `_ := expr?` discards it and propagates only the failure. A bare `expr?` statement
// is rejected. The propagation mode (Result vs Option) is taken from the enclosing
// function's return type — `?` early-returns the same kind the function returns.
//
// Scope: OPEN-E only (E is `error`). Closed-E `?` needs the From-conversion (feature
// 06) and is out of scope. No error checking, no type inference. Malformed input is
// undefined behavior.
package main

import (
	"fmt"
	"go/format"
	"sort"
	"strconv"
	"strings"
	"text/scanner"
	"unicode"
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

type replacement struct {
	start, end int
	text       string
}

// propagation mode of a function, from its return type.
type fnMode int

const (
	modeNone fnMode = iota
	modeResult
	modeOption
)

type fnInfo struct {
	bodyStart, bodyEnd int
	mode               fnMode
}

// hygienic temporaries / named returns (§8 prefix).
const (
	okName  = "__gop_ok"
	errName = "__gop_err"
	optBase = "__gop_o"
)

// transpile lowers goal source using `?` (plus the Result/Option forms it needs).
func transpile(src string) (string, error) {
	toks := lex(src)
	var reps []replacement

	// Pass 1: scan functions — rewrite Result return types to named returns and
	// record each function's propagation mode for the `?` pass.
	funcs := scanFuncs(src, toks, &reps)

	// Pass 2: `Option[T]` -> `*T` wherever it appears (including Option returns).
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].text == "Option" && toks[i+1].text == "[" {
			rb := matchBracket(toks, i+1)
			t := strings.TrimSpace(src[toks[i+2].start:toks[rb].start])
			reps = append(reps, replacement{toks[i].start, toks[rb].end, "*" + t})
			i = rb
		}
	}

	// Pass 3: construction in return position (Result.Ok/Err, Option.Some/None).
	reps = append(reps, lowerReturns(src, toks)...)

	// Pass 4: `?` propagation.
	qReps, err := lowerQuestion(src, toks, funcs)
	if err != nil {
		return "", err
	}
	reps = append(reps, qReps...)

	out := splice(src, 0, len(src), reps)
	formatted, err := format.Source([]byte(out))
	if err != nil {
		return "", fmt.Errorf("generated Go did not parse: %w\n--- generated ---\n%s", err, out)
	}
	return string(formatted), nil
}

// scanFuncs records each function's body span and propagation mode, and appends a
// signature rewrite (to named returns) for every Result[T, error] function.
func scanFuncs(src string, toks []token, reps *[]replacement) []fnInfo {
	var funcs []fnInfo
	for i := range toks {
		if toks[i].text != "func" {
			continue
		}
		bo := firstBodyBrace(toks, i)
		if bo < 0 {
			continue
		}
		bc := matchBrace(toks, bo)
		pc := paramsClose(toks, bo)
		info := fnInfo{bodyStart: toks[bo].start, bodyEnd: toks[bc].end, mode: modeNone}
		if pc >= 0 && pc+2 < bo {
			switch {
			case toks[pc+1].text == "Result" && toks[pc+2].text == "[":
				rb := matchBracket(toks, pc+2)
				comma := topLevelComma(toks, pc+2, rb)
				if comma >= 0 {
					t := strings.TrimSpace(src[toks[pc+3].start:toks[comma].start])
					*reps = append(*reps, replacement{toks[pc+1].start, toks[rb].end,
						fmt.Sprintf("(%s %s, %s error)", okName, t, errName)})
					info.mode = modeResult
				}
			case toks[pc+1].text == "Option" && toks[pc+2].text == "[":
				info.mode = modeOption // type rewritten by Pass 2
			}
		}
		funcs = append(funcs, info)
	}
	return funcs
}

// lowerReturns lowers `return Result.Ok/Err(...)` and `return Option.Some/None`.
func lowerReturns(src string, toks []token) []replacement {
	var reps []replacement
	for i := 0; i+3 < len(toks); i++ {
		if toks[i].text != "return" || toks[i+2].text != "." {
			continue
		}
		switch toks[i+1].text {
		case "Result":
			if toks[i+4].text != "(" {
				continue
			}
			closeIdx := matchParen(toks, i+4)
			inner := strings.TrimSpace(src[toks[i+4].end:toks[closeIdx].start])
			var text string
			if toks[i+3].text == "Ok" {
				text = inner + ", nil"
			} else {
				text = okName + ", " + inner
			}
			reps = append(reps, replacement{toks[i+1].start, toks[closeIdx].end, text})
			i = closeIdx
		case "Option":
			switch toks[i+3].text {
			case "None":
				reps = append(reps, replacement{toks[i+1].start, toks[i+3].end, "nil"})
				i += 3
			case "Some":
				if toks[i+4].text != "(" {
					continue
				}
				closeIdx := matchParen(toks, i+4)
				x := strings.TrimSpace(src[toks[i+4].end:toks[closeIdx].start])
				var text string
				if closeIdx == i+6 && isIdent(toks[i+5].text) {
					text = "return &" + x
				} else {
					text = fmt.Sprintf("%s := %s\nreturn &%s", optBase, x, optBase)
				}
				reps = append(reps, replacement{toks[i].start, toks[closeIdx].end, text})
				i = closeIdx
			}
		}
	}
	return reps
}

// lowerQuestion lowers each `?`. A `?` must terminate `name := expr?` or `_ := expr?`
// (a bare `expr?` is rejected). The mode comes from the enclosing function.
func lowerQuestion(src string, toks []token, funcs []fnInfo) ([]replacement, error) {
	var reps []replacement
	optCounter := 0
	for q := range toks {
		if toks[q].text != "?" {
			continue
		}
		p := toks[q].start
		lineStart := strings.LastIndexByte(src[:p], '\n') + 1
		name, rhs, ok := splitAssign(src[lineStart:p])
		if !ok {
			return nil, fmt.Errorf("`?` must be the right-hand side of an assignment; write `name := expr?` to keep the value or `_ := expr?` to discard it")
		}
		discard := name == "_"
		var text string
		switch modeAt(funcs, p) {
		case modeResult:
			if discard {
				text = fmt.Sprintf("if _, %s := %s; %s != nil {\nreturn %s, %s\n}", errName, rhs, errName, okName, errName)
			} else {
				text = fmt.Sprintf("%s, %s := %s\nif %s != nil {\nreturn %s, %s\n}", name, errName, rhs, errName, okName, errName)
			}
		case modeOption:
			optCounter++
			o := optBase + strconv.Itoa(optCounter)
			if discard {
				text = fmt.Sprintf("if %s := %s; %s == nil {\nreturn nil\n}", o, rhs, o)
			} else {
				text = fmt.Sprintf("%s := %s\nif %s == nil {\nreturn nil\n}\n%s := *%s", o, rhs, o, name, o)
			}
		default:
			return nil, fmt.Errorf("`?` outside a Result- or Option-returning function (open-E only; closed-E `?` is feature 06)")
		}
		reps = append(reps, replacement{lineStart, toks[q].end, text})
	}
	return reps, nil
}

// splitAssign parses `lhs := rhs` (or just `rhs`). ok is false when there is no `:=`.
func splitAssign(s string) (name, rhs string, ok bool) {
	if lhs, after, found := strings.Cut(s, ":="); found {
		return strings.TrimSpace(lhs), strings.TrimSpace(after), true
	}
	return "", strings.TrimSpace(s), false
}

func modeAt(funcs []fnInfo, off int) fnMode {
	for _, f := range funcs {
		if off >= f.bodyStart && off < f.bodyEnd {
			return f.mode
		}
	}
	return modeNone
}

// ----- shared helpers -----

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

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	r := []rune(s)[0]
	return unicode.IsLetter(r) || r == '_'
}

func matchParen(toks []token, openIdx int) int   { return matchPair(toks, openIdx, "(", ")") }
func matchBracket(toks []token, openIdx int) int { return matchPair(toks, openIdx, "[", "]") }
func matchBrace(toks []token, openIdx int) int   { return matchPair(toks, openIdx, "{", "}") }

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

func splice(src string, lo, hi int, reps []replacement) string {
	sort.Slice(reps, func(a, b int) bool { return reps[a].start < reps[b].start })
	var b strings.Builder
	prev := lo
	for _, r := range reps {
		if r.start < prev {
			continue
		}
		b.WriteString(src[prev:r.start])
		b.WriteString(r.text)
		prev = r.end
	}
	b.WriteString(src[prev:hi])
	return b.String()
}
