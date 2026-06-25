// Package main is a standalone reference transpiler for goal feature 02-match:
// pattern-matching `match` with exhaustiveness, lowered to a Go type-switch over
// the closed sum-type encoding from feature 01 (spec §8.2). It also lowers the
// `enum` declarations the matches refer to (the §8.1 encoding, reused from 01) so
// that each example file produces self-contained, compilable Go.
//
// Scope: this transpiler ASSUMES well-formed, type-correct, EXHAUSTIVE input. It
// does NO error checking — no exhaustiveness verification, no closedness check, no
// type inference. Per the audit prompt it lowers proven-exhaustive matches to the
// panic-default; an explicit `_` arm becomes a real default. Malformed input is
// undefined behavior.
//
// match positions handled: statement, `return match`, and `var name T = match`.
// The untyped `name := match` value form needs the checker's inferred result type
// and is intentionally deferred (see transpileMatch's error path).
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

// lex tokenizes goal source. goal is a Go dialect, so a generic identifier/
// string/number lexer suffices: keywords (enum, match, return, var) lex as
// identifiers and punctuation (. : { } ( ) , => as = >) lex as single-rune
// tokens. Whitespace and comments live only in the gaps between tokens and are
// recovered from the original source via byte offsets when passing text through.
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

type field struct {
	name string // goal field name (lowercase)
	typ  string // type expression, verbatim source
}

type variant struct {
	name   string
	fields []field // empty => data-less
}

type enumInfo struct {
	name     string
	variants []variant
	fieldSet map[string]map[string]bool // variant -> set of field names
}

// replacement is a byte span of the source to splice over with generated Go.
type replacement struct {
	start, end int
	text       string
}

// transpile lowers goal source (enum declarations + match expressions) into
// idiomatic Go, then gofmt-formats the result.
func transpile(src string) (string, error) {
	toks := lex(src)
	enums := map[string]*enumInfo{}
	var reps []replacement

	// Pass 1: enum declarations build the registry (variant field names feed the
	// match lowering's binding rewrites) and lower to the §8.1 encoding.
	for i := 0; i < len(toks); {
		if toks[i].text == "enum" {
			rep, info, next := parseEnum(src, toks, i)
			reps = append(reps, rep)
			enums[info.name] = info
			i = next
			continue
		}
		i++
	}

	// Pass 2: lower each match to a type-switch (§8.2).
	for i := 0; i < len(toks); {
		if toks[i].text == "match" {
			rep, next, err := lowerMatch(src, toks, enums, i)
			if err != nil {
				return "", err
			}
			reps = append(reps, rep)
			i = next
			continue
		}
		i++
	}

	out := splice(src, 0, len(src), reps)
	formatted, err := format.Source([]byte(out))
	if err != nil {
		return "", fmt.Errorf("generated Go did not parse: %w\n--- generated ---\n%s", err, out)
	}
	return string(formatted), nil
}

// ----- enum lowering (the §8.1 encoding, reused from feature 01) -----

func parseEnum(src string, toks []token, i int) (replacement, *enumInfo, int) {
	name := toks[i+1].text
	k := i + 3 // step past `enum NAME {`
	info := &enumInfo{name: name, fieldSet: map[string]map[string]bool{}}
	for toks[k].text != "}" {
		vname := toks[k].text
		k++
		var fields []field
		if toks[k].text == "{" {
			fields, k = parseFields(src, toks, k+1)
			k++ // consume variant's closing "}"
		}
		info.variants = append(info.variants, variant{name: vname, fields: fields})
		set := map[string]bool{}
		for _, f := range fields {
			set[f.name] = true
		}
		info.fieldSet[vname] = set
	}
	return replacement{toks[i].start, toks[k].end, genEnum(info)}, info, k + 1
}

func parseFields(src string, toks []token, k int) ([]field, int) {
	var fields []field
	for toks[k].text != "}" {
		name := toks[k].text
		k += 2 // skip name and ":"
		typeStart := toks[k].start
		typeEnd := toks[k].end
		depth := 0
		for {
			t := toks[k]
			if depth == 0 && (t.text == "," || t.text == "}") {
				break
			}
			switch t.text {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			}
			typeEnd = t.end
			k++
		}
		fields = append(fields, field{name: name, typ: strings.TrimSpace(src[typeStart:typeEnd])})
		if toks[k].text == "," {
			k++
		}
	}
	return fields, k
}

func genEnum(info *enumInfo) string {
	marker := "is" + info.name
	var b strings.Builder
	fmt.Fprintf(&b, "type %s interface{ %s() }\n\n", info.name, marker)
	for _, v := range info.variants {
		if len(v.fields) == 0 {
			fmt.Fprintf(&b, "type %s_%s struct{}\n", info.name, v.name)
			continue
		}
		fmt.Fprintf(&b, "type %s_%s struct {\n", info.name, v.name)
		for _, f := range v.fields {
			fmt.Fprintf(&b, "\t%s %s\n", exported(f.name), f.typ)
		}
		b.WriteString("}\n")
	}
	b.WriteString("\n")
	for _, v := range info.variants {
		fmt.Fprintf(&b, "func (%s_%s) %s() {}\n", info.name, v.name, marker)
	}
	return b.String()
}

// ----- match lowering (§8.2) -----

// matchPos is where a match sits: as a statement, in return position, or as the
// initializer of an explicitly-typed `var`.
type matchPos int

const (
	posStmt matchPos = iota
	posReturn
	posVar
)

// vbind is the hygienic name for the type-switch guard variable (§8 prefix).
const vbind = "__goal_v"

type arm struct {
	rest    bool   // the `_` rest-arm
	enum    string // for non-rest arms
	variant string
	binding string // "" if no payload binding
	bodyLo  int    // token index of first body token
	bodyHi  int    // token index just past last body token
}

// lowerMatch lowers the match whose `match` keyword is at toks[mi]. It returns the
// replacement (covering the whole match, including any leading `return`/`var`), the
// token index to resume scanning from, and an error for unsupported forms.
func lowerMatch(src string, toks []token, enums map[string]*enumInfo, mi int) (replacement, int, error) {
	pos, name, typ, repStart, err := classifyPosition(src, toks, mi)
	if err != nil {
		return replacement{}, 0, err
	}

	// Scrutinee: text between `match` and the arm-block `{`.
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

	arms := parseArms(toks, bo+1, bc)

	// Lower each arm body first so we know whether the guard variable is used.
	bodies := make([]string, len(arms))
	usesBinding := false
	for i, a := range arms {
		var used bool
		bodies[i], used = rewriteBody(src, toks, a, enums)
		usesBinding = usesBinding || used
	}

	var b strings.Builder
	if pos == posVar {
		fmt.Fprintf(&b, "var %s %s\n", name, typ)
	}
	if usesBinding {
		fmt.Fprintf(&b, "switch %s := %s.(type) {\n", vbind, scrut)
	} else {
		fmt.Fprintf(&b, "switch %s.(type) {\n", scrut)
	}

	enumName := ""
	restBody := ""
	hasRest := false
	for i, a := range arms {
		if a.rest {
			hasRest = true
			restBody = bodies[i]
			continue
		}
		enumName = a.enum
		fmt.Fprintf(&b, "case %s_%s:\n", a.enum, a.variant)
		b.WriteString(armStatement(pos, name, bodies[i]))
	}

	b.WriteString("default:\n")
	if hasRest {
		b.WriteString(armStatement(pos, name, restBody))
	} else {
		fmt.Fprintf(&b, "panic(%q)\n", fmt.Sprintf("unreachable: non-exhaustive %s (compiler invariant violated)", enumName))
	}
	b.WriteString("}")

	// For posVar the `var NAME TYPE` line already precedes the switch; the user's
	// code after the original declaration continues to use NAME.
	return replacement{repStart, toks[bc].end, b.String()}, bc + 1, nil
}

// classifyPosition inspects the tokens before `match` to decide the position and,
// for posVar, recover the declared name and type. repStart is where the generated
// replacement begins (the `return`/`var` keyword, or `match` itself).
func classifyPosition(src string, toks []token, mi int) (pos matchPos, name, typ string, repStart int, err error) {
	if mi == 0 {
		return posStmt, "", "", toks[mi].start, nil
	}
	prev := toks[mi-1].text
	switch {
	case prev == "return":
		return posReturn, "", "", toks[mi-1].start, nil
	case prev == "=" && mi-2 >= 0 && toks[mi-2].text == ":":
		return 0, "", "", 0, fmt.Errorf("value-position `name := match` needs the checker's inferred result type (deferred in the reference transpiler); use `var name T = match ...` or `return match ...`")
	case prev == "=":
		return classifyVar(src, toks, mi)
	default:
		return posStmt, "", "", toks[mi].start, nil
	}
}

// classifyVar handles `var NAME TYPE = match ...` by walking back to the `var`.
func classifyVar(src string, toks []token, mi int) (matchPos, string, string, int, error) {
	eq := mi - 1 // the "=" token
	k := eq - 1
	for depth := 0; k >= 0; k-- {
		switch toks[k].text {
		case ")", "]", "}":
			depth++
		case "(", "[", "{":
			depth--
		}
		if depth == 0 && toks[k].text == "var" {
			break
		}
	}
	if k < 0 || toks[k].text != "var" {
		return 0, "", "", 0, fmt.Errorf("match in `= match` position must be a `var NAME TYPE = match ...` declaration")
	}
	name := toks[k+1].text
	typ := strings.TrimSpace(src[toks[k+2].start:toks[eq].start])
	return posVar, name, typ, toks[k].start, nil
}

// armStatement wraps a lowered arm body for the match position.
func armStatement(pos matchPos, name, body string) string {
	switch pos {
	case posReturn:
		return "return " + body + "\n"
	case posVar:
		return name + " = " + body + "\n"
	default:
		return body + "\n"
	}
}

// matchBrace returns the index of the "}" matching the "{" at openIdx.
func matchBrace(toks []token, openIdx int) int {
	depth := 0
	for k := openIdx; k < len(toks); k++ {
		switch toks[k].text {
		case "{":
			depth++
		case "}":
			depth--
		}
		if depth == 0 {
			return k
		}
	}
	return len(toks) - 1
}

// parseArms splits the arm-block tokens [lo, hi) into arms. Arms are delimited by
// `=>` (lexed as `=` `>`): the tokens just before each arrow form that arm's
// pattern, and everything between one arrow and the next arm's pattern is the body.
func parseArms(toks []token, lo, hi int) []arm {
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

	arms := make([]arm, len(arrows))
	for i, eq := range arrows {
		var patStart int
		if i == 0 {
			patStart = lo
		} else {
			patStart = patternStart(toks, arrows[i])
		}
		a := parsePattern(toks, patStart, eq)
		a.bodyLo = eq + 2 // skip "=" ">"
		if i+1 < len(arrows) {
			a.bodyHi = patternStart(toks, arrows[i+1])
		} else {
			a.bodyHi = hi
		}
		arms[i] = a
	}
	return arms
}

// patternStart finds where the arm pattern ending just before the arrow at eqIdx
// begins, by inspecting the token immediately before the arrow.
func patternStart(toks []token, eqIdx int) int {
	j := eqIdx - 1
	switch toks[j].text {
	case ")":
		// Enum . Variant ( binding ) — walk back to the "(" then to the Enum.
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
	case "_":
		return j
	default:
		// Enum . Variant
		return j - 2
	}
}

func parsePattern(toks []token, start, eqIdx int) arm {
	if toks[start].text == "_" {
		return arm{rest: true}
	}
	a := arm{enum: toks[start].text, variant: toks[start+2].text}
	if start+3 < eqIdx && toks[start+3].text == "(" {
		a.binding = toks[start+4].text
	}
	return a
}

// rewriteBody returns the arm body source with the payload binding rewritten to the
// guard variable and field accesses on it exported (since -> Since). The bool
// reports whether the binding was referenced (so the caller knows if the guard
// variable is needed).
func rewriteBody(src string, toks []token, a arm, enums map[string]*enumInfo) (string, bool) {
	lo, hi := a.bodyLo, a.bodyHi
	if lo >= hi {
		return "", false
	}
	used := false
	var fields map[string]bool
	if a.enum != "" {
		if e, ok := enums[a.enum]; ok {
			fields = e.fieldSet[a.variant]
		}
	}

	var reps []replacement
	for j := lo; j < hi; {
		if a.binding != "" && toks[j].text == a.binding {
			used = true
			if j+2 < hi && toks[j+1].text == "." && fields[toks[j+2].text] {
				reps = append(reps, replacement{toks[j].start, toks[j].end, vbind})
				f := toks[j+2]
				reps = append(reps, replacement{f.start, f.end, exported(f.text)})
				j += 3
				continue
			}
			reps = append(reps, replacement{toks[j].start, toks[j].end, vbind})
		}
		j++
	}
	return splice(src, toks[lo].start, toks[hi-1].end, reps), used
}

// exported capitalizes the first rune so a goal field name maps to an exported Go field.
func exported(name string) string {
	if name == "" {
		return name
	}
	r := []rune(name)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
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
