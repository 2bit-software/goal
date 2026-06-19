// Package main is a standalone reference transpiler for goal feature 07-implements:
// explicit `implements X for T` — a struct declaring it satisfies (at least)
// interface X. The assertion is checked at the declaration site by the goal checker
// and then ERASED (§3.4, §8.5); the only generated Go is the free, runtime-cost-free
// compile-time assertion `var _ X = T{}` (recommended by §8.5), which keeps the output
// self-verifying and documents intent.
//
// The `implements X for T` surface was pinned in feature 01 (the sealed-interface
// form); this feature reuses it for the general additive assertion over any interface.
//
// Scope: the reference transpiler EMITS the assertion; it does NOT verify the methods
// exist or match (the checker's job). If T's methods use a pointer receiver, the
// assertion is emitted as `var _ X = (*T)(nil)` so it still compiles. Malformed input
// is undefined behavior.
package main

import (
	"fmt"
	"go/format"
	"sort"
	"strings"
	"text/scanner"
	"unicode"
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

// transpile lowers `implements X for T` to the §8.5 compile-time assertion.
func transpile(src string) (string, error) {
	toks := lex(src)

	// Pass 1: record which types have at least one pointer-receiver method, so the
	// assertion can address the type correctly (only *T satisfies X then).
	pointerRecv := scanPointerReceivers(toks)

	// Pass 2: replace each `implements X for T` declaration with the assertion.
	var reps []replacement
	for i := range toks {
		if toks[i].text != "implements" {
			continue
		}
		j := i + 1
		for j < len(toks) && toks[j].text != "for" {
			j++
		}
		if j >= len(toks) {
			continue
		}
		iface := strings.TrimSpace(src[toks[i].end:toks[j].start])
		lineEnd := len(src)
		if nl := strings.IndexByte(src[toks[j].end:], '\n'); nl >= 0 {
			lineEnd = toks[j].end + nl
		}
		typ := strings.TrimSpace(src[toks[j].end:lineEnd])
		base := baseType(typ)

		var assertion string
		if pointerRecv[base] {
			assertion = fmt.Sprintf("var _ %s = (*%s)(nil)", iface, base)
		} else {
			assertion = fmt.Sprintf("var _ %s = %s{}", iface, typ)
		}
		reps = append(reps, replacement{toks[i].start, lineEnd, assertion})
	}

	out := splice(src, reps)
	formatted, err := format.Source([]byte(out))
	if err != nil {
		return "", fmt.Errorf("generated Go did not parse: %w\n--- generated ---\n%s", err, out)
	}
	return string(formatted), nil
}

// scanPointerReceivers returns the set of type names that have at least one method
// with a pointer receiver (`func (x *T) ...`).
func scanPointerReceivers(toks []token) map[string]bool {
	set := map[string]bool{}
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].text != "func" || toks[i+1].text != "(" {
			continue
		}
		rc := matchParen(toks, i+1)
		star := false
		typeName := ""
		for k := i + 2; k < rc; k++ {
			switch {
			case toks[k].text == "*":
				star = true
			case isIdent(toks[k].text):
				typeName = toks[k].text // last identifier in the receiver = the type
			}
		}
		if star && typeName != "" {
			set[typeName] = true
		}
	}
	return set
}

// baseType strips a leading `*` and any `pkg.` qualifier, yielding the receiver type
// name to look up (the implemented type is always local, hence unqualified).
func baseType(t string) string {
	t = strings.TrimSpace(strings.TrimPrefix(t, "*"))
	if i := strings.LastIndexByte(t, '.'); i >= 0 {
		t = t[i+1:]
	}
	return t
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	r := []rune(s)[0]
	return unicode.IsLetter(r) || r == '_'
}

func matchParen(toks []token, openIdx int) int {
	depth := 0
	for k := openIdx; k < len(toks); k++ {
		switch toks[k].text {
		case "(":
			depth++
		case ")":
			depth--
		}
		if depth == 0 {
			return k
		}
	}
	return len(toks) - 1
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
