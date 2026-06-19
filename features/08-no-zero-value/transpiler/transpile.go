// Package main is a standalone reference transpiler for goal feature
// 08-no-zero-value: required-field struct construction. In goal, constructing a
// struct requires every field be set explicitly (or explicitly defaulted);
// forgetting a field is a located compile error, not a silent Go zero value
// (§3.5). That check is ERASED at codegen — the feature "only ever rejected
// source" (§8.5), so a complete struct literal lowers to itself, verbatim.
//
// The one thing this transpiler DOES rewrite is the explicit-defaults escape
// hatch `...defaults` (the spelling chosen for this feature). It names the
// common "I really do want the zeros for the rest" case, and lowers to explicit
// per-field zero values so the generated Go literal is also complete (§8.5):
//
//	User{name: n, role: RoleMember, ...defaults}
//	  ->  User{name: n, role: RoleMember, email: "", active: false, logins: 0}
//
// Scope (NO checking — the checker's job, not built here): this transpiler does
// NOT reject incomplete literals, does NOT verify field names, and does NOT
// decide whether a default is semantically appropriate (e.g. an enum has no safe
// zero — the checker rejects that; here `...defaults` mechanically expands it).
// It only expands `...defaults` against the struct's declared fields, recovering
// each field's zero from its declared type. Field types whose zero is not
// syntactically recoverable (func/chan types with internal spaces, grouped
// `type ( … )` decls) are out of scope; malformed input is undefined behavior.
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

// field is one declared struct field: its name and the raw text of its type.
type field struct {
	name string
	typ  string
}

// transpile expands every `...defaults` form into explicit per-field zero values
// and passes everything else (including complete struct literals) through verbatim.
func transpile(src string) (string, error) {
	toks := lex(src)
	structFields, decls := parseTypeDecls(src, toks)

	var reps []replacement
	for i := range toks {
		if !isDefaultsForm(toks, i) {
			continue
		}
		// The form occupies tokens `.` `.` `.` `defaults`, i.e. toks[i-3..i].
		spanStart, spanEnd := toks[i-3].start, toks[i].end

		openIdx := enclosingBrace(toks, i-4)
		if openIdx < 1 {
			return "", fmt.Errorf("`...defaults` at offset %d is not inside a struct literal", spanStart)
		}
		typeName := toks[openIdx-1].text
		fields, ok := structFields[typeName]
		if !ok {
			return "", fmt.Errorf("`...defaults` for unknown struct type %q (no `type %s struct{…}` in this file)", typeName, typeName)
		}

		present := presentFields(toks, openIdx)
		var entries []string
		for _, f := range fields {
			if present[f.name] {
				continue
			}
			entries = append(entries, fmt.Sprintf("%s: %s", f.name, zeroLit(f.typ, decls, 0)))
		}
		reps = append(reps, replacement{spanStart, spanEnd, strings.Join(entries, ", ")})
	}

	out := splice(src, reps)
	formatted, err := format.Source([]byte(out))
	if err != nil {
		return "", fmt.Errorf("generated Go did not parse: %w\n--- generated ---\n%s", err, out)
	}
	return string(formatted), nil
}

// isDefaultsForm reports whether toks[i] is the `defaults` ident of a `...defaults`
// element (three leading `.` tokens).
func isDefaultsForm(toks []token, i int) bool {
	return i >= 3 && toks[i].text == "defaults" &&
		toks[i-1].text == "." && toks[i-2].text == "." && toks[i-3].text == "."
}

// enclosingBrace scans backward from `from` to find the `{` of the composite
// literal that directly contains it (depth-aware). Returns -1 if none.
func enclosingBrace(toks []token, from int) int {
	depth := 0
	for k := from; k >= 0; k-- {
		switch toks[k].text {
		case "}", "]", ")":
			depth++
		case "{", "[", "(":
			if depth == 0 {
				if toks[k].text == "{" {
					return k
				}
				return -1
			}
			depth--
		}
	}
	return -1
}

// presentFields collects the keyed field names already set in the literal whose
// opening brace is at openIdx (keys at the literal's own depth: `IDENT :`).
func presentFields(toks []token, openIdx int) map[string]bool {
	present := map[string]bool{}
	close := matchBrace(toks, openIdx)
	depth := 0
	for k := openIdx + 1; k < close; k++ {
		switch toks[k].text {
		case "{", "[", "(":
			depth++
		case "}", "]", ")":
			depth--
		}
		if depth == 0 && isIdent(toks[k].text) && k+1 < close && toks[k+1].text == ":" {
			present[toks[k].text] = true
		}
	}
	return present
}

// parseTypeDecls scans top-level `type` declarations, returning the ordered field
// list of every `type X struct {…}` and a name->underlying map for resolving the
// zero value of named types (`struct`, `interface`, an alias target, or a defined
// type's underlying type expression).
func parseTypeDecls(src string, toks []token) (map[string][]field, map[string]string) {
	structFields := map[string][]field{}
	decls := map[string]string{}
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].text != "type" || !isIdent(toks[i+1].text) {
			continue
		}
		name := toks[i+1].text
		switch toks[i+2].text {
		case "=":
			decls[name] = restOfLine(src, toks[i+3].start)
		case "struct":
			decls[name] = "struct"
			open := indexOf(toks, i+2, "{")
			if open >= 0 {
				close := matchBrace(toks, open)
				structFields[name] = parseStructBody(src[toks[open].end:toks[close].start])
			}
		case "interface":
			decls[name] = "interface"
		default:
			decls[name] = restOfLine(src, toks[i+2].start)
		}
	}
	return structFields, decls
}

// parseStructBody parses the text between a struct's braces into ordered fields.
// One field per line (or `;`-separated); `a, b int` yields two fields.
func parseStructBody(body string) []field {
	var fields []field
	for _, raw := range strings.FieldsFunc(body, func(r rune) bool { return r == '\n' || r == ';' }) {
		line := raw
		if c := strings.Index(line, "//"); c >= 0 {
			line = line[:c]
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue // blank, or an embedded type (unsupported here) — skip
		}
		typ := parts[len(parts)-1]
		for _, nm := range parts[:len(parts)-1] {
			nm = strings.TrimSuffix(nm, ",")
			if nm != "" {
				fields = append(fields, field{name: nm, typ: typ})
			}
		}
	}
	return fields
}

// zeroLit returns the explicit Go zero literal for a declared field type. Untyped
// constants (`0`, `""`, `false`) are assignable to defined types, so a defined
// type like `type Role int` correctly defaults to `0`. depth guards alias chains.
func zeroLit(typ string, decls map[string]string, depth int) string {
	typ = strings.TrimSpace(typ)
	switch {
	case strings.HasPrefix(typ, "*"), strings.HasPrefix(typ, "[]"),
		strings.HasPrefix(typ, "map["), strings.HasPrefix(typ, "chan"),
		strings.HasPrefix(typ, "func"), strings.HasPrefix(typ, "interface"),
		typ == "any", typ == "error":
		return "nil"
	case strings.HasPrefix(typ, "["): // array `[N]T` — composite zero
		return typ + "{}"
	}
	switch typ {
	case "string":
		return `""`
	case "bool":
		return "false"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"byte", "rune", "float32", "float64", "complex64", "complex128":
		return "0"
	}
	if depth < 8 {
		if under, ok := decls[baseType(typ)]; ok {
			switch under {
			case "struct":
				return typ + "{}"
			case "interface":
				return "nil"
			default:
				return zeroLit(under, decls, depth+1)
			}
		}
	}
	// Unknown named type with no local declaration: assume a struct-like composite
	// zero. (A named interface from another package would want `nil`; not
	// recoverable without a type system — out of scope, noted in TRANSPILE.md.)
	return typ + "{}"
}

// restOfLine returns the source from offset to the next newline, trimmed (and with
// a leading `=` stripped so an alias `type X = Y` yields just `Y`).
func restOfLine(src string, offset int) string {
	end := len(src)
	if nl := strings.IndexByte(src[offset:], '\n'); nl >= 0 {
		end = offset + nl
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(src[offset:end]), "="))
}

// indexOf returns the index of the first token with text t at or after `from`.
func indexOf(toks []token, from int, t string) int {
	for k := from; k < len(toks); k++ {
		if toks[k].text == t {
			return k
		}
	}
	return -1
}

// baseType strips a leading `*` and any `pkg.` qualifier.
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
