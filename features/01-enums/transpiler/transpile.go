// Package main is a standalone reference transpiler for goal feature 01-enums:
// closed sum types (real enums) and their two surface forms, lowered to the
// sealed-interface + per-variant-struct + unexported-marker Go encoding (spec §8.1).
//
// Scope: this transpiler ASSUMES well-formed, type-correct input. It does NO
// error checking (no exhaustiveness, no closedness verification, no field checks)
// — those are the checker's job (FEATURE-AUDIT-PROMPT.md Step 3). Malformed input
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

// token is a lexical token with its byte span in the source.
type token struct {
	text  string
	start int
	end   int
}

// lex tokenizes goal source. goal is a Go dialect, so a generic identifier/
// string/number lexer suffices: keywords (enum, sealed, implements, for) lex as
// identifiers and punctuation (. : { } ( ) , [ ]) lex as single-rune tokens.
// Whitespace and comments live only in the gaps between tokens and are recovered
// from the original source via byte offsets when passing text through.
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
	vset     map[string]bool
}

// replacement is a byte span of the source to splice over with generated Go.
type replacement struct {
	start, end int
	text       string
}

// transpile lowers goal source containing enum/sealed/implements declarations and
// variant constructions into idiomatic Go, then gofmt-formats the result.
func transpile(src string) (string, error) {
	toks := lex(src)
	enums := map[string]*enumInfo{}
	var reps []replacement

	// Pass 1: declarations (enum, sealed interface, struct implements clause) build the
	// registry and emit their encodings.
	for i := 0; i < len(toks); {
		switch toks[i].text {
		case "enum":
			rep, info, next := parseEnum(src, toks, i)
			reps = append(reps, rep)
			enums[info.name] = info
			i = next
		case "sealed":
			rep, next := parseSealed(toks, i)
			reps = append(reps, rep)
			i = next
		case "type":
			rs, next := parseStructImplements(src, toks, i)
			reps = append(reps, rs...)
			i = next
		default:
			i++
		}
	}

	// Pass 2: rewrite variant constructions NAME.V[(...)] now that all enums are known.
	for j := 0; j+2 < len(toks); {
		e, ok := enums[toks[j].text]
		if ok && toks[j+1].text == "." && e.vset[toks[j+2].text] {
			vname := toks[j+2].text
			if j+3 < len(toks) && toks[j+3].text == "(" {
				closeIdx := matchParen(toks, j+3)
				args := parseArgs(src, toks, j+4, closeIdx)
				reps = append(reps, replacement{toks[j].start, toks[closeIdx].end, construct(e.name, vname, args)})
				j = closeIdx + 1
				continue
			}
			reps = append(reps, replacement{toks[j].start, toks[j+2].end, construct(e.name, vname, nil)})
			j += 3
			continue
		}
		j++
	}

	out := splice(src, reps)
	formatted, err := format.Source([]byte(out))
	if err != nil {
		return "", fmt.Errorf("generated Go did not parse: %w\n--- generated ---\n%s", err, out)
	}
	return string(formatted), nil
}

// parseEnum parses `enum NAME { variant... }` starting at toks[i] == "enum".
// Returns the replacement (whole decl -> generated Go), the parsed info, and the
// index just past the closing brace.
func parseEnum(src string, toks []token, i int) (replacement, *enumInfo, int) {
	name := toks[i+1].text
	k := i + 2 // toks[k] == "{"
	k++        // step into body
	info := &enumInfo{name: name, vset: map[string]bool{}}
	for toks[k].text != "}" {
		vname := toks[k].text
		k++
		var fields []field
		if toks[k].text == "{" {
			fields, k = parseFields(src, toks, k+1) // k -> variant's closing "}"
			k++                                     // consume it
		}
		info.variants = append(info.variants, variant{name: vname, fields: fields})
		info.vset[vname] = true
	}
	end := toks[k].end // closing brace of enum body
	return replacement{toks[i].start, end, genEnum(info)}, info, k + 1
}

// parseFields parses `name: Type, name: Type` up to the closing "}", starting at
// toks[k] (the first field name). Type expressions are captured verbatim from
// source, honoring nested () [] {} so map/func/struct/slice types survive intact.
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

// parseSealed parses `sealed interface NAME {}` starting at toks[i] == "sealed".
func parseSealed(toks []token, i int) (replacement, int) {
	name := toks[i+2].text // toks[i+1] == "interface"
	// toks[i+3] == "{", toks[i+4] == "}"
	end := toks[i+4].end
	return replacement{toks[i].start, end, genInterface(name)}, i + 5
}

// parseStructImplements handles a struct declaration that may carry an inline
// `implements` clause — `type T struct implements X, Y { … }`. When present, it returns
// two replacements (collapse the clause; emit a marker method per interface after the
// struct's closing brace) and the index past the struct. A plain struct (or any other
// `type` form) yields no replacements and advances by one token.
func parseStructImplements(src string, toks []token, i int) ([]replacement, int) {
	if i+2 >= len(toks) || !isIdent(toks[i+1].text) || toks[i+2].text != "struct" {
		return nil, i + 1
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
		return nil, i + 1
	}
	imp := -1
	for k := i + 3; k < open; k++ {
		if toks[k].text == "implements" {
			imp = k
			break
		}
	}
	if imp < 0 {
		return nil, i + 1
	}
	closeIdx := matchBrace(toks, open)

	var b strings.Builder
	for _, iface := range splitInterfaces(src[toks[imp].end:toks[open].start]) {
		b.WriteString(genMarker(name, iface))
		b.WriteByte('\n')
	}
	if b.Len() == 0 {
		return nil, closeIdx + 1
	}
	return []replacement{
		{toks[i+2].end, toks[open].start, " "},
		{toks[closeIdx].end, toks[closeIdx].end, "\n\n" + b.String()},
	}, closeIdx + 1
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

// isIdent reports whether s begins like a Go identifier (letter or underscore).
func isIdent(s string) bool {
	if s == "" {
		return false
	}
	r := []rune(s)[0]
	return unicode.IsLetter(r) || r == '_'
}

type kv struct {
	label string
	expr  string
}

// matchParen returns the index of the ")" matching the "(" at openIdx.
func matchParen(toks []token, openIdx int) int { return matchPair(toks, openIdx, "(", ")") }

// matchBrace returns the index of the "}" matching the "{" at openIdx.
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

// parseArgs parses `label: expr, label: expr` between an open paren (exclusive,
// start = first arg token) and closeIdx (the matching ")"). Expressions are
// captured verbatim, honoring nesting so calls like now() survive.
func parseArgs(src string, toks []token, start, closeIdx int) []kv {
	var args []kv
	k := start
	for k < closeIdx {
		label := toks[k].text
		k += 2 // skip label and ":"
		exprStart := toks[k].start
		exprEnd := toks[k].end
		depth := 0
		for k < closeIdx {
			t := toks[k]
			if depth == 0 && t.text == "," {
				break
			}
			switch t.text {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			}
			exprEnd = t.end
			k++
		}
		args = append(args, kv{label: label, expr: strings.TrimSpace(src[exprStart:exprEnd])})
		if k < closeIdx && toks[k].text == "," {
			k++
		}
	}
	return args
}

// genEnum emits the §8.1 encoding for a single-block enum.
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

func genInterface(name string) string {
	return fmt.Sprintf("type %s interface{ is%s() }", name, name)
}

func genMarker(typ, iface string) string {
	return fmt.Sprintf("func (%s) is%s() {}", typ, iface)
}

// construct emits a variant construction: NAME(NAME_V{Field: expr, ...}).
func construct(enum, variant string, args []kv) string {
	if len(args) == 0 {
		return fmt.Sprintf("%s(%s_%s{})", enum, enum, variant)
	}
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = fmt.Sprintf("%s: %s", exported(a.label), a.expr)
	}
	return fmt.Sprintf("%s(%s_%s{%s})", enum, enum, variant, strings.Join(parts, ", "))
}

// exported capitalizes the first rune so a goal field/label maps to an exported Go field.
func exported(name string) string {
	if name == "" {
		return name
	}
	r := []rune(name)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// splice rebuilds the source with each replacement span swapped for its generated
// text, leaving all other bytes (including comments and whitespace) untouched.
func splice(src string, reps []replacement) string {
	sort.Slice(reps, func(a, b int) bool { return reps[a].start < reps[b].start })
	var b strings.Builder
	prev := 0
	for _, r := range reps {
		if r.start < prev {
			continue // defensive: skip any overlap
		}
		b.WriteString(src[prev:r.start])
		b.WriteString(r.text)
		prev = r.end
	}
	b.WriteString(src[prev:])
	return b.String()
}
