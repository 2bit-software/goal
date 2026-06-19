// Package main is a standalone reference transpiler for goal feature 04-option:
// `Option[T]` / nil-safety, lowered with the §8.4 pointer strategy. `Option[T]` ->
// `*T`; `Option.None` -> `nil`; `Option.Some(v)` -> `&v` (or a boxed temp when v is
// not a plain addressable identifier); a `match` on the Option lowers to the
// idiomatic `if p := ...; p != nil { x := *p; ... } else { ... }`.
//
// Scope: handles the IMMEDIATE case (§8.7) — an Option returned, and match-ed at the
// use site. Value types (Option[int]) box to *int (v1, §8.4). Construction is only
// recognized in `return` position; a stored Option value and value-position Option
// match are deferred. This transpiler does NO error checking (no must-use, no
// exhaustiveness, no type inference). Malformed input is undefined behavior.
package main

import (
	"fmt"
	"go/format"
	"sort"
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

// hygienic temporaries (§8 prefix).
const (
	optName  = "__gop_o"    // the captured pointer at a match site
	someName = "__gop_some" // boxed Some value when the payload isn't addressable
)

// transpile lowers goal source using Option[T] into idiomatic Go.
func transpile(src string) (string, error) {
	toks := lex(src)
	var reps []replacement

	// Pass A: rewrite the type `Option[T]` -> `*T` wherever it appears.
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].text == "Option" && toks[i+1].text == "[" {
			rb := matchBracket(toks, i+1)
			t := strings.TrimSpace(src[toks[i+2].start:toks[rb].start])
			reps = append(reps, replacement{toks[i].start, toks[rb].end, "*" + t})
			i = rb
		}
	}

	// Pass B: lower `return Option.None` and `return Option.Some(X)`.
	for i := 0; i+3 < len(toks); i++ {
		if toks[i].text != "return" || toks[i+1].text != "Option" || toks[i+2].text != "." {
			continue
		}
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
				text = "return &" + x // addressable identifier -> &x directly (§8.4)
			} else {
				text = fmt.Sprintf("%s := %s\nreturn &%s", someName, x, someName) // box
			}
			reps = append(reps, replacement{toks[i].start, toks[closeIdx].end, text})
			i = closeIdx
		}
	}

	// Pass C: lower statement-position `match opt { Option.Some/None arms }`.
	for i := 0; i < len(toks); i++ {
		if toks[i].text == "match" {
			rep, next, err := lowerOptionMatch(src, toks, i)
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

// ----- pass C: Option match -----

type optionArm struct {
	variant string // "Some" or "None"
	binding string // "" if none
	bodyLo  int
	bodyHi  int
}

func lowerOptionMatch(src string, toks []token, mi int) (replacement, int, error) {
	if mi > 0 {
		if p := toks[mi-1].text; p == "return" || p == "=" {
			return replacement{}, 0, fmt.Errorf("value-position Option match is deferred in the reference transpiler; consume an Option with a statement-position match (§8.4)")
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
	arms := parseOptionArms(toks, bo+1, bc)

	var some, none *optionArm
	for i := range arms {
		switch arms[i].variant {
		case "Some":
			some = &arms[i]
		case "None":
			none = &arms[i]
		}
	}
	if some == nil || none == nil {
		return replacement{}, 0, fmt.Errorf("Option match must have both Option.Some and Option.None arms")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "if %s := %s; %s != nil {\n", optName, scrut, optName)
	if some.binding != "" && bodyUses(toks, some.bodyLo, some.bodyHi, some.binding) {
		fmt.Fprintf(&b, "%s := *%s\n", some.binding, optName)
	}
	b.WriteString(bodySrc(src, toks, some.bodyLo, some.bodyHi))
	b.WriteString("\n} else {\n")
	b.WriteString(bodySrc(src, toks, none.bodyLo, none.bodyHi))
	b.WriteString("\n}")
	return replacement{toks[mi].start, toks[bc].end, b.String()}, bc + 1, nil
}

func parseOptionArms(toks []token, lo, hi int) []optionArm {
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

	arms := make([]optionArm, len(arrows))
	for i, eq := range arrows {
		patStart := lo
		if i > 0 {
			patStart = patternStart(toks, arrows[i])
		}
		a := optionArm{variant: toks[patStart+2].text}
		if patStart+3 < eq && toks[patStart+3].text == "(" {
			a.binding = toks[patStart+4].text
		}
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

// patternStart finds where the `Option.Variant[(binding)]` pattern ending just
// before the arrow at eqIdx begins.
func patternStart(toks []token, eqIdx int) int {
	j := eqIdx - 1
	if toks[j].text == ")" {
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
		return k - 3 // Option . Variant ( binding )
	}
	return j - 2 // Option . Variant
}

func bodySrc(src string, toks []token, lo, hi int) string {
	if lo >= hi {
		return ""
	}
	return strings.TrimSpace(src[toks[lo].start:toks[hi-1].end])
}

func bodyUses(toks []token, lo, hi int, name string) bool {
	for j := lo; j < hi; j++ {
		if toks[j].text == name {
			return true
		}
	}
	return false
}

// ----- shared helpers -----

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
