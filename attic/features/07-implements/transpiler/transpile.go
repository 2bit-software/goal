// Package main is a standalone reference transpiler for goal feature 07-implements:
// the inline `implements` clause on a struct — `type T struct implements X, Y { … }` —
// a struct declaring it satisfies (at least) interfaces X and Y. The assertion is
// checked at the declaration site by the goal checker and then ERASED (§3.4, §8.5); the
// only generated Go is the free, runtime-cost-free compile-time assertion
// `var _ X = T{}` (recommended by §8.5), which keeps the output self-verifying and
// documents intent.
//
// Only struct types carry the clause today; extending it to any concrete type (as Go
// allows, e.g. `type Celsius float64 implements Stringer`) is future work.
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

// transpile lowers a struct's inline `implements` clause to the §8.5 compile-time
// assertion, emitted just after the struct's closing brace.
func transpile(src string) (string, error) {
	toks := lex(src)

	// Pass 1: record which types have at least one pointer-receiver method, so the
	// assertion can address the type correctly (only *T satisfies X then).
	pointerRecv := scanPointerReceivers(toks)

	// Pass 2: for each `type T struct implements X, Y { … }`, strip the clause and emit
	// one assertion per interface after the struct.
	var reps []replacement
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].text != "type" || !isIdent(toks[i+1].text) || toks[i+2].text != "struct" {
			continue
		}
		name := toks[i+1].text
		open := -1
		for k := i + 3; k < len(toks); k++ {
			if toks[k].text == "{" {
				open = k
				break
			}
		}
		if open < 0 {
			continue
		}
		imp := -1
		for k := i + 3; k < open; k++ {
			if toks[k].text == "implements" {
				imp = k
				break
			}
		}
		if imp < 0 {
			continue // a plain struct with no implements clause
		}
		closeIdx := matchBrace(toks, open)

		var b strings.Builder
		for _, iface := range splitInterfaces(src[toks[imp].end:toks[open].start]) {
			if pointerRecv[name] {
				fmt.Fprintf(&b, "var _ %s = (*%s)(nil)", iface, name)
			} else {
				fmt.Fprintf(&b, "var _ %s = %s{}", iface, name)
			}
			b.WriteByte('\n')
		}
		if b.Len() == 0 {
			continue
		}
		reps = append(reps, replacement{toks[i+2].end, toks[open].start, " "})
		reps = append(reps, replacement{toks[closeIdx].end, toks[closeIdx].end, "\n\n" + b.String()})
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

// splitInterfaces splits a clause's comma-separated interface list into trimmed names,
// dropping empties. Qualified names (`io.Writer`) survive intact — they carry no comma.
func splitInterfaces(s string) []string {
	var out []string
	for part := range strings.SplitSeq(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	r := []rune(s)[0]
	return unicode.IsLetter(r) || r == '_'
}

func matchParen(toks []token, openIdx int) int { return matchPair(toks, openIdx, "(", ")") }

func matchBrace(toks []token, openIdx int) int { return matchPair(toks, openIdx, "{", "}") }

// matchPair returns the index of the close delimiter matching the open delimiter at
// openIdx.
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
