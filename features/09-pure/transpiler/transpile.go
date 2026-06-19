// Package main is a standalone reference transpiler for goal feature 09-pure:
// the lightweight `pure func` annotation (§4.2). `pure` is a declarable-and-checked
// "this function has no side effects" marker — NOT a granular effect system. The
// checker verifies the absence of effects in a `pure` body; that guarantee is then
// ERASED at codegen (§8.5): `pure func` lowers to a plain `func`, nothing else.
//
//	pure func square(x int) int { return x * x }
//	  ->  func square(x int) int { return x * x }
//
// Scope (NO checking — the checker's job, not built here): this transpiler does
// NOT verify that a `pure` body is actually effect-free. It only strips the `pure`
// keyword wherever it modifies a `func` (free function or method). Malformed input
// is undefined behavior. The marker is the same shape as feature 06's `from func`,
// so the lowering mirrors it: strip the modifier, pass everything else through.
package main

import (
	"fmt"
	"go/format"
	"sort"
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

// transpile erases the `pure` modifier from every `pure func …` declaration and
// passes everything else through verbatim.
func transpile(src string) (string, error) {
	toks := lex(src)

	var reps []replacement
	for i := 0; i+1 < len(toks); i++ {
		// `pure` is the marker only when it directly modifies a `func` (free
		// function or method). Anywhere else it is an ordinary identifier and is
		// left untouched.
		if toks[i].text == "pure" && toks[i+1].text == "func" {
			// Strip `pure ` — from the `pure` token up to the `func` token — so
			// `pure func` becomes `func`.
			reps = append(reps, replacement{toks[i].start, toks[i+1].start, ""})
		}
	}

	out := splice(src, reps)
	formatted, err := format.Source([]byte(out))
	if err != nil {
		return "", fmt.Errorf("generated Go did not parse: %w\n--- generated ---\n%s", err, out)
	}
	return string(formatted), nil
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
