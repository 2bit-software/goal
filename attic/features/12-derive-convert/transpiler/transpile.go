// Package main is a standalone reference transpiler for goal feature
// 12-derive-convert: type-directed, completeness-checked struct conversion.
//
// Two constructs (see SYNTAX.md / TRANSPILE.md):
//
//   - Leaf conversion `from func NAME(p Src) Ret` — generalizes feature 06's
//     `from func` to any type pair. Every in-scope leaf is a registry entry keyed
//     by (Src -> Tgt). Tier is read from Ret: `(Tgt, error)` is fallible (tier 3),
//     a plain `Tgt` is total (tier 1, or tier 2 if it asserts internally). The
//     `from` keyword is stripped so the leaf becomes a plain Go func.
//
//   - Derived conversion `derive func NAME(src S) T [body]` — the compiler fills
//     the body field-by-field, resolving each target field through the registry.
//     `...derive(src)` fills the unmentioned fields (the parallel of feature 08's
//     `...defaults`); `Field: expr` is a verbatim override; `Field: _` skips.
//
// Scope (NO full checker — per the audit's no-checking-yet constraint): the
// transpiler resolves what it can and DEFERS an unresolvable field with a located
// error (never silently zero — that is the footgun this feature kills). Slice
// container recursion is implemented; map/Option/nested-struct recursion follow the
// same rule but are minimal in v1. Examples use lowered Go forms (`(T, error)`,
// `*string`, local UUID/NullString) so this transpiler needs no dependency on
// features 03/04. Malformed input is UB.
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

type field struct {
	name string
	typ  string
}

// convEntry is one registry conversion: the function name and whether it is fallible.
type convEntry struct {
	name     string
	fallible bool
}

func convKey(src, tgt string) string { return strings.TrimSpace(src) + "\x00" + strings.TrimSpace(tgt) }

// transpile strips `from` from leaf conversions, builds the registry, and expands
// every `derive func` into idiomatic field-by-field Go.
func transpile(src string) (string, error) {
	toks := lex(src)

	registry, fromReps := buildRegistry(src, toks)
	structFields := parseStructs(src, toks)

	deriveReps, err := expandDerives(src, toks, registry, structFields)
	if err != nil {
		return "", err
	}

	out := splice(src, append(fromReps, deriveReps...))
	formatted, ferr := format.Source([]byte(out))
	if ferr != nil {
		return "", fmt.Errorf("generated Go did not parse: %w\n--- generated ---\n%s", ferr, out)
	}
	return string(formatted), nil
}

// buildRegistry records every `from func` as a (src -> tgt) conversion and returns
// the replacements that strip the leading `from ` (so each leaf becomes plain Go).
func buildRegistry(src string, toks []token) (map[string]convEntry, []replacement) {
	reg := map[string]convEntry{}
	var reps []replacement
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].text != "from" || toks[i+1].text != "func" {
			continue
		}
		open := indexOf(toks, i+2, "(")
		if open < 0 {
			continue
		}
		closeP := matchPair(toks, open, "(", ")")
		paramType := strings.TrimSpace(src[toks[open+1].end:toks[closeP].start])
		retType := strings.TrimSpace(src[toks[closeP].end:firstBraceAfter(src, toks[closeP].end)])
		tgt, fallible := parseReturn(retType)
		reg[convKey(paramType, tgt)] = convEntry{name: toks[i+2].text, fallible: fallible}
		reps = append(reps, replacement{toks[i].start, toks[i+1].start, ""}) // strip `from `
	}
	return reg, reps
}

// parseReturn splits a return type into its target type and whether it is fallible
// (`(T, error)` -> fallible with target T; a bare `T` -> total).
func parseReturn(ret string) (tgt string, fallible bool) {
	ret = strings.TrimSpace(ret)
	if strings.HasPrefix(ret, "(") && strings.HasSuffix(ret, ")") {
		inner := ret[1 : len(ret)-1]
		first, _, _ := strings.Cut(inner, ",")
		return strings.TrimSpace(first), true
	}
	return ret, false
}

// expandDerives replaces each `derive func` declaration with a generated function.
func expandDerives(src string, toks []token, reg map[string]convEntry, structFields map[string][]field) ([]replacement, error) {
	var reps []replacement
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].text != "derive" || toks[i+1].text != "func" {
			continue
		}
		name := toks[i+2].text
		open := indexOf(toks, i+2, "(")
		closeP := matchPair(toks, open, "(", ")")
		srcName := toks[open+1].text
		srcType := strings.TrimSpace(src[toks[open+1].end:toks[closeP].start])

		// Return type runs to the body `{` or, if bodyless, to end of line.
		afterParams := toks[closeP].end
		brace := strings.IndexByte(src[afterParams:], '{')
		nl := strings.IndexByte(src[afterParams:], '\n')
		hasBody := brace >= 0 && (nl < 0 || brace < nl)

		var retType string
		var overrides []field // name -> expr text ("_" => skip)
		var declEnd int
		if hasBody {
			bodyOpen := afterParams + brace
			retType = strings.TrimSpace(src[afterParams:bodyOpen])
			bodyOpenTok := tokenAt(toks, bodyOpen)
			bodyClose := matchPair(toks, bodyOpenTok, "{", "}")
			declEnd = toks[bodyClose].end
			overrides = parseOverrides(src, toks, bodyOpenTok, bodyClose)
		} else {
			end := len(src)
			if nl >= 0 {
				end = afterParams + nl
			}
			retType = strings.TrimSpace(src[afterParams:end])
			declEnd = end
		}

		tgtType, fallible := parseReturn(retType)
		gen, err := genConversion(name, srcName, srcType, tgtType, retType, fallible, overrides, reg, structFields)
		if err != nil {
			return nil, err
		}
		reps = append(reps, replacement{toks[i].start, declEnd, gen})
	}
	return reps, nil
}

// parseOverrides reads `Field: expr` / `Field: _` entries from a derive body's
// returned composite literal (ignoring the `...derive(src)` element).
func parseOverrides(src string, toks []token, openIdx, closeIdx int) []field {
	// The literal brace is the first `{` after `return`.
	ret := indexOf(toks, openIdx+1, "return")
	litOpen := indexOf(toks, ret+1, "{")
	if litOpen < 0 || litOpen >= closeIdx {
		return nil
	}
	litClose := matchPair(toks, litOpen, "{", "}")
	var out []field
	depth := 0
	for k := litOpen + 1; k < litClose; k++ {
		switch toks[k].text {
		case "{", "[", "(":
			depth++
		case "}", "]", ")":
			depth--
		}
		if depth == 0 && isIdent(toks[k].text) && k+1 < litClose && toks[k+1].text == ":" {
			name := toks[k].text
			valStart := toks[k+2].start
			valEnd := topLevelCommaOrClose(toks, k+2, litClose)
			expr := strings.TrimSpace(src[valStart:valEnd])
			out = append(out, field{name: name, typ: expr})
		}
	}
	return out
}

// genConversion produces the Go function body for one derived conversion.
func genConversion(name, srcName, srcType, tgtType, retType string, fallible bool, overrides []field, reg map[string]convEntry, structFields map[string][]field) (string, error) {
	tgtFields, ok := structFields[tgtType]
	if !ok {
		return "", fmt.Errorf("derive %s: unknown target struct %q", name, tgtType)
	}
	srcFields := structFields[srcType]

	overridden := map[string]string{}
	for _, o := range overrides {
		overridden[strings.ToLower(o.name)] = o.typ
	}

	var b strings.Builder
	fmt.Fprintf(&b, "func %s(%s %s) %s {\n", name, srcName, srcType, retType)
	b.WriteString("var out " + tgtType + "\n")

	// Explicit overrides first, in written order (`_` => leave zero).
	for _, o := range overrides {
		if strings.TrimSpace(o.typ) == "_" {
			continue
		}
		fmt.Fprintf(&b, "out.%s = %s\n", o.name, o.typ)
	}

	// `...derive(src)`: every remaining target field, registry-resolved.
	tempN := 0
	for _, f := range tgtFields {
		if _, done := overridden[strings.ToLower(f.name)]; done {
			continue
		}
		sf, found := findField(srcFields, f.name)
		if !found {
			return "", fmt.Errorf("derive %s: target field %q of %s is not sourced from %s (add an explicit `%s: …` or a `from func`)", name, f.name, tgtType, srcType, f.name)
		}
		stmts, err := resolve("out."+f.name, srcName+"."+sf.name, sf.typ, f.typ, reg, fallible, &tempN)
		if err != nil {
			return "", fmt.Errorf("derive %s, field %q: %w", name, f.name, err)
		}
		for _, s := range stmts {
			b.WriteString(s + "\n")
		}
	}

	if fallible {
		b.WriteString("return out, nil\n}")
	} else {
		b.WriteString("return out\n}")
	}
	return b.String(), nil
}

// resolve emits the statements that assign a converted source field to a target
// field, choosing the strategy by (source type -> target type).
func resolve(dst, srcExpr, sf, tf string, reg map[string]convEntry, fallibleOK bool, tempN *int) ([]string, error) {
	sf, tf = strings.TrimSpace(sf), strings.TrimSpace(tf)
	if sf == tf {
		return []string{fmt.Sprintf("%s = %s", dst, srcExpr)}, nil
	}
	if e, ok := reg[convKey(sf, tf)]; ok {
		if !e.fallible {
			return []string{fmt.Sprintf("%s = %s(%s)", dst, e.name, srcExpr)}, nil
		}
		if !fallibleOK {
			return nil, fmt.Errorf("conversion %s->%s is fallible; declare the derive func returning (T, error)", sf, tf)
		}
		v := fmt.Sprintf("__goal_v%d", *tempN)
		*tempN++
		return []string{
			fmt.Sprintf("%s, err := %s(%s)", v, e.name, srcExpr),
			"if err != nil {\nreturn out, err\n}",
			fmt.Sprintf("%s = %s", dst, v),
		}, nil
	}
	// Built-in container recursion: []A -> []B when A -> B resolves (total only, v1).
	if strings.HasPrefix(sf, "[]") && strings.HasPrefix(tf, "[]") {
		elem, err := elemConv(sf[2:], tf[2:], reg)
		if err != nil {
			return nil, err
		}
		return []string{
			fmt.Sprintf("%s = make(%s, len(%s))", dst, tf, srcExpr),
			fmt.Sprintf("for i := range %s {\n%s = %s\n}", srcExpr, dst+"[i]", elem(srcExpr+"[i]")),
		}, nil
	}
	return nil, fmt.Errorf("no conversion %s -> %s in scope", sf, tf)
}

// elemConv returns a function that renders the conversion of a single slice element
// expression from type a to type b (total conversions only in v1).
func elemConv(a, b string, reg map[string]convEntry) (func(string) string, error) {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == b {
		return func(x string) string { return x }, nil
	}
	if e, ok := reg[convKey(a, b)]; ok && !e.fallible {
		return func(x string) string { return e.name + "(" + x + ")" }, nil
	}
	return nil, fmt.Errorf("no total element conversion %s -> %s for slice recursion", a, b)
}

// parseStructs returns the ordered fields of every `type X struct {…}`.
func parseStructs(src string, toks []token) map[string][]field {
	out := map[string][]field{}
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].text != "type" || toks[i+2].text != "struct" {
			continue
		}
		open := indexOf(toks, i+2, "{")
		if open < 0 {
			continue
		}
		closeB := matchPair(toks, open, "{", "}")
		out[toks[i+1].text] = parseStructBody(src[toks[open].end:toks[closeB].start])
	}
	return out
}

func parseStructBody(body string) []field {
	var fields []field
	for _, raw := range strings.FieldsFunc(body, func(r rune) bool { return r == '\n' || r == ';' }) {
		line := raw
		if c := strings.Index(line, "//"); c >= 0 {
			line = line[:c]
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		typ := parts[len(parts)-1]
		for _, nm := range parts[:len(parts)-1] {
			if nm = strings.TrimSuffix(nm, ","); nm != "" {
				fields = append(fields, field{name: nm, typ: typ})
			}
		}
	}
	return fields
}

func findField(fields []field, name string) (field, bool) {
	for _, f := range fields {
		if strings.EqualFold(f.name, name) {
			return f, true
		}
	}
	return field{}, false
}

// ----- small token helpers -----

func indexOf(toks []token, from int, t string) int {
	for k := from; k < len(toks); k++ {
		if toks[k].text == t {
			return k
		}
	}
	return -1
}

func tokenAt(toks []token, offset int) int {
	for k := range toks {
		if toks[k].start == offset {
			return k
		}
	}
	return -1
}

func firstBraceAfter(src string, offset int) int {
	if b := strings.IndexByte(src[offset:], '{'); b >= 0 {
		return offset + b
	}
	return len(src)
}

// topLevelCommaOrClose returns the byte offset of the first top-level `,` (or the
// closing brace) bounding a composite-literal element value. Only a top-level comma
// terminates the value — interior `.` (method calls like e.ID.String()) do not.
func topLevelCommaOrClose(toks []token, from, closeIdx int) int {
	depth := 0
	for k := from; k < closeIdx; k++ {
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
	return toks[closeIdx].start
}

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

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	r := []rune(s)[0]
	return unicode.IsLetter(r) || r == '_'
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
