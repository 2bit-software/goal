package check

import (
	"fmt"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// checkFields enforces feature 08 (no-zero-value): every struct or enum-variant
// composite literal must name a value for every field, unless it opts out with the
// `...defaults` spread. A literal that omits a field without the spread is an Error.
//
// Two surface forms carry per-field payloads and are checked here:
//
//   - Struct literals `T{…}` (keyed composite literals). Completeness may be satisfied
//     explicitly or via a complete-by-construction spread — `...defaults` (§8.5, the
//     defaults pass fills the rest) or `...derive(src)` (§12, the derive pass fills the
//     rest field-by-field); an omission without such a spread is an Error. This is the
//     documented common case.
//   - Enum variant constructions `Enum.Variant(field: expr, …)` (paren form, the surface
//     the enums pass lowers). A variant has no `...defaults` escape, so every declared
//     field must be named; an omission is an Error.
//
// Reuse, not reinvention:
//   - The defaults pass (internal/pass/defaults.go) locates each `T{…}` literal and its
//     present keys (enclosingBrace / presentFields logic) — mirrored here as
//     structLitKeys, but asserting completeness instead of expanding.
//   - Required field sets: analyze.Tables.Structs (ordered fields per struct) and
//     analyze.Tables.Enums[…].FieldSet (field-name set per variant).
//   - Match arm-pattern spans (matchPatternSpans) mirror the match pass's arm locator
//     (internal/pass/match.go parseMatchArms + internal/pass/result.go patternStart): a
//     `Enum.Variant(a)` inside an arm pattern is a binding, NOT a construction, so it is
//     skipped — the same `=>`/patternStart machinery the lowering uses to read arms.
//
// Defer-boundary: this check only fires when the literal's type is named *at the site*
// — `T{…}` where T is a known struct, or `Enum.Variant(…)` where Enum is a known enum.
// A composite literal with no type name at the site (a bare `{…}` element of a typed
// outer literal, an array/slice/map element, a `:=` whose type is not at the site) is
// surfaced as a located Warning naming the unresolved literal, never assumed — a false
// "complete" is worse than an honest "cannot tell here".
func checkFields(src string, t *analyze.Tables) ([]Diagnostic, error) {
	toks := scan.Lex(src)
	declSpans := declBraceSpans(toks)
	bodyBraces := funcBodyBraces(toks)
	ctrlBraces := controlBodyBraces(toks)
	patSpans := matchPatternSpans(toks)
	var diags []Diagnostic
	for i := 0; i+1 < len(toks); i++ {
		// Struct literal: IDENT "{" where IDENT names a known struct. Three braces wear
		// the same `IDENT {` shape but are not literals: a type-declaration body
		// (`enum E { … }`, `type T struct { … }`), a function body whose `{` follows the
		// return type (`func f() User {`), and a control-flow body whose `{` follows a
		// tag/condition that ends in a bare identifier (`switch i {`, `if ok {`,
		// `for more {`). Skip all three.
		// A fifth shape is the element/value TYPE of a slice/array/map literal
		// (`[]Entry{…}`, `[N]Entry{…}`, `map[K]Entry{…}`): the type name is immediately
		// preceded by `]` and the `{` opens that outer literal, not a struct — no valid
		// struct literal is preceded by `] IDENT {`. Skip it.
		if toks[i+1].Text == "{" && scan.IsIdent(toks[i].Text) &&
			(i == 0 || toks[i-1].Text != "]") &&
			!inSpans(i+1, declSpans) && !bodyBraces[i+1] && !ctrlBraces[i+1] {
			if ds, ok := checkStructLit(src, toks, i, t); ok {
				diags = append(diags, ds...)
			}
			continue
		}
		// Enum variant construction: Enum "." Variant "(" where Enum is a known enum.
		// A `Enum.Variant(a)` sitting in a match arm-pattern position is a payload
		// BINDING (`Status.Active(a) => …`), not a construction — it binds the narrowed
		// value to a name and must not be checked for field completeness. Skip any site
		// whose qualifier token falls inside an arm-pattern span.
		if toks[i+1].Text == "." && i+3 < len(toks) && toks[i+3].Text == "(" &&
			!inSpans(i, patSpans) {
			if ds, ok := checkVariantLit(src, toks, i, t); ok {
				diags = append(diags, ds...)
			}
		}
	}
	return diags, nil
}

// checkStructLit checks the composite literal opening at toks[i] == typeName, toks[i+1]
// == "{". ok is false when toks[i] is not a struct-literal type name to act on (so the
// caller moves on). When the named type is an in-file struct it asserts completeness;
// when it is a non-struct type name with a brace block (an enum/interface/alias) it is
// not a struct literal and is skipped; an unknown capitalized type name that looks like
// a constructed literal is deferred with a Warning.
func checkStructLit(src string, toks []scan.Token, i int, t *analyze.Tables) ([]Diagnostic, bool) {
	typeName := toks[i].Text
	// Skip the language keywords that take a brace block but are not struct literals
	// (`struct{}`, `interface{}`, `map[…]{…}` handled via the `[` guard below, control
	// flow, etc.). A struct literal's type token is a plain identifier, never a keyword.
	if isBraceKeyword(typeName) {
		return nil, false
	}
	// `T[...]{...}` (generic) and `pkg.T{...}` (qualified) put a `]`/`.` between name and
	// brace, so a bare `IDENT {` with no qualifier is what we see; a qualified or generic
	// literal is deferred below by the not-in-Structs path.
	fields, known := t.Structs[typeName]
	if !known {
		// Only treat this as a (deferred) struct literal when it actually reads like a
		// keyed composite literal — the brace contains `IDENT :` at its own depth.
		// Otherwise it is a block, a func body, a map/slice literal element, etc., and
		// is none of our business.
		if _, hasKeys := litKeys(toks, i+1); !hasKeys {
			return nil, false
		}
		// A keyed literal whose type we cannot resolve in-file: defer, don't guess.
		return []Diagnostic{{
			Pos:      toks[i].Start,
			Severity: Warning,
			Feature:  "08-no-zero-value",
			Code:     "unresolved-literal-type",
			Message: fmt.Sprintf("cannot verify field completeness of `%s{…}`: type `%s` is not declared in this file — field-completeness deferred",
				typeName, typeName),
		}}, true
	}
	present, _ := litKeys(toks, i+1)
	if litHasCompletingSpread(toks, i+1) {
		// `...defaults` opts the rest into their zeros; `...derive(src)` opts the rest
		// into derived values. Either spread makes the literal complete by construction —
		// the defaults / derive pass owns expanding (and rejecting) the unnamed fields.
		return nil, true
	}
	missing := missingFields(fields, present)
	if len(missing) == 0 {
		return nil, true
	}
	return []Diagnostic{{
		Pos:      toks[i].Start,
		Severity: Error,
		Feature:  "08-no-zero-value",
		Code:     "missing-field",
		Message: fmt.Sprintf("struct literal `%s{…}` omits required field%s %s — set %s explicitly, or add `...defaults` to fill the rest with zero values",
			typeName, plural(len(missing)), quoteJoin(missing), pronoun(len(missing))),
	}}, true
}

// checkVariantLit checks an enum variant construction Enum.Variant(...) opening at
// toks[i] == Enum, toks[i+1] == ".", toks[i+2] == Variant, toks[i+3] == "(". ok is
// false when toks[i] is not a known enum (the caller moves on). A variant has no
// `...defaults` escape, so every declared field must be named; a payload-less variant
// (empty field set) is trivially complete.
func checkVariantLit(src string, toks []scan.Token, i int, t *analyze.Tables) ([]Diagnostic, bool) {
	enum, ok := t.Enums[toks[i].Text]
	if !ok {
		return nil, false
	}
	vname := toks[i+2].Text
	fieldSet, isVariant := enum.FieldSet[vname]
	if !isVariant {
		return nil, false
	}
	if len(fieldSet) == 0 {
		return nil, true // data-less variant: nothing to complete.
	}
	close := scan.MatchParen(toks, i+3)
	present := parenKeys(toks, i+3, close)
	var declared []analyze.Field
	for _, v := range enum.Variants {
		if v.Name == vname {
			declared = v.Fields
			break
		}
	}
	missing := missingFields(declared, present)
	if len(missing) == 0 {
		return nil, true
	}
	return []Diagnostic{{
		Pos:      toks[i].Start,
		Severity: Error,
		Feature:  "08-no-zero-value",
		Code:     "missing-field",
		Message: fmt.Sprintf("variant construction `%s.%s(…)` omits required field%s %s — a variant has no `...defaults`; name every field",
			enum.Name, vname, plural(len(missing)), quoteJoin(missing)),
	}}, true
}

// span is a half-open token-index range [open, close] of a declaration body brace.
type span struct{ open, close int }

// declBraceSpans returns the brace-body spans of every `enum E { … }` and
// `type T struct { … }` declaration. A `{` inside one of these is part of a type
// declaration — its `name: Type` / `name Type` field lines are NOT a composite
// literal's keyed elements and must be skipped by the literal scan. (Variant field
// braces like `Circle { … }` nest inside the enum span, so the single outer span
// covers them.)
func declBraceSpans(toks []scan.Token) []span {
	var spans []span
	for i := 0; i+1 < len(toks); i++ {
		isEnum := toks[i].Text == "enum"
		isStruct := toks[i].Text == "struct"
		if !isEnum && !isStruct {
			continue
		}
		// Find this declaration's opening brace: `enum NAME {` or `struct {`.
		open := -1
		for k := i + 1; k < len(toks) && k <= i+3; k++ {
			if toks[k].Text == "{" {
				open = k
				break
			}
		}
		if open < 0 {
			continue
		}
		spans = append(spans, span{open: open, close: scan.MatchBrace(toks, open)})
	}
	return spans
}

// funcBodyBraces returns the set of token indices that are a function/method body's
// opening "{". Such a brace follows the return type as `IDENT {` and would otherwise be
// misread as a composite literal of that return type.
func funcBodyBraces(toks []scan.Token) map[int]bool {
	bodies := map[int]bool{}
	for _, f := range scan.ScanFuncs(toks) {
		if f.BodyOpen >= 0 {
			bodies[f.BodyOpen] = true
		}
	}
	return bodies
}

// controlBodyBraces returns the set of token indices that open a control-flow statement
// body — the "{" after a `switch`/`if`/`for`/`select` header. Such a brace follows the
// header expression as `<header> {` and, when the header ends in a bare identifier
// (`switch i {`, `if ok {`, `for more {`), would otherwise be misread as a composite
// literal of that identifier's "type", deferring on a phantom type. A composite literal
// in one of these headers must be parenthesized per the grammar, so the body is the first
// "{" at paren/bracket depth 0 after the keyword. The lone exception is a `for … range
// []T{…} {` range clause, whose element-type literal is the first depth-0 "{"; mistaking
// it for the body only forgoes a deferral on a non-struct type, never reports an error.
func controlBodyBraces(toks []scan.Token) map[int]bool {
	bodies := map[int]bool{}
	for i := range toks {
		switch toks[i].Text {
		case "switch", "if", "for", "select":
			if b := scan.FirstBodyBrace(toks, i); b >= 0 {
				bodies[b] = true
			}
		}
	}
	return bodies
}

// inSpans reports whether token index idx falls strictly inside any declaration span.
func inSpans(idx int, spans []span) bool {
	for _, s := range spans {
		if idx > s.open && idx < s.close {
			return true
		}
	}
	return false
}

// litKeys returns the keyed field names set at the literal's own brace depth (an
// `IDENT :` at depth 0 inside the braces, mirroring the defaults pass presentFields),
// and whether the brace block contained any such key. openIdx is the index of the "{".
func litKeys(toks []scan.Token, openIdx int) (map[string]bool, bool) {
	present := map[string]bool{}
	closeIdx := scan.MatchBrace(toks, openIdx)
	depth := 0
	any := false
	for k := openIdx + 1; k < closeIdx; k++ {
		switch toks[k].Text {
		case "{", "[", "(":
			depth++
		case "}", "]", ")":
			depth--
		}
		if depth == 0 && scan.IsIdent(toks[k].Text) && k+1 < closeIdx && toks[k+1].Text == ":" {
			present[toks[k].Text] = true
			any = true
		}
	}
	return present, any
}

// parenKeys returns the labelled argument names of a variant construction
// `(label: expr, …)` at its own paren depth, between openIdx (the "(") and closeIdx.
func parenKeys(toks []scan.Token, openIdx, closeIdx int) map[string]bool {
	present := map[string]bool{}
	depth := 0
	for k := openIdx + 1; k < closeIdx; k++ {
		switch toks[k].Text {
		case "{", "[", "(":
			depth++
		case "}", "]", ")":
			depth--
		}
		if depth == 0 && scan.IsIdent(toks[k].Text) && k+1 < closeIdx && toks[k+1].Text == ":" {
			present[toks[k].Text] = true
		}
	}
	return present
}

// litHasCompletingSpread reports whether the literal whose "{" is at openIdx contains a
// complete-by-construction spread element at its own brace depth: `...defaults` (§8.5,
// the four-token `.` `.` `.` `defaults` the defaults pass recognizes) or `...derive`
// (§12, `.` `.` `.` `derive` followed by `(src)`, the four-token spread the derive pass
// recognizes — internal/pass/derive.go). Either spread means the unnamed fields are
// filled by that owning pass, so completeness here is not this check's to assert.
func litHasCompletingSpread(toks []scan.Token, openIdx int) bool {
	closeIdx := scan.MatchBrace(toks, openIdx)
	depth := 0
	for k := openIdx + 1; k < closeIdx; k++ {
		switch toks[k].Text {
		case "{", "[", "(":
			depth++
		case "}", "]", ")":
			depth--
		}
		if depth == 0 && k >= openIdx+4 &&
			(toks[k].Text == "defaults" || toks[k].Text == "derive") &&
			toks[k-1].Text == "." && toks[k-2].Text == "." && toks[k-3].Text == "." {
			return true
		}
	}
	return false
}

// matchPatternSpans returns, for every `match` arm block, the half-open token-index
// spans `{open, close}` that an arm PATTERN occupies (so a `Enum.Variant(a)` inside one
// is a payload binding, not a construction). It mirrors the match pass's arm locator:
// the arm block is `MatchBodyBrace`…`MatchBrace`, arms are delimited by depth-0 `=>`
// (lexed as `=` then `>`), and each arm's pattern runs from `patternStart(arrow)` up to
// the arrow. Spans are encoded for `inSpans` (strict `open < idx < close`) by widening
// `open` to `patternStart-1` so the qualifier token itself reads as inside.
//
// This claims arm patterns of EVERY match — not only enum matches — because the false
// positive being closed is purely lexical: a single-identifier bind `(a)` in pattern
// position reads as a variant construction with no keyed fields. Matches whose first
// token after `{` is not part of a `Qual.Variant(...)` pattern simply yield no variant
// site for the caller to check, so over-claiming pattern spans is harmless.
func matchPatternSpans(toks []scan.Token) []span {
	var spans []span
	for mi := 0; mi < len(toks); mi++ {
		if toks[mi].Text != "match" {
			continue
		}
		bo := scan.MatchBodyBrace(toks, mi)
		if bo < 0 {
			continue
		}
		bc := scan.MatchBrace(toks, bo)
		// Arrows: depth-0 `=` immediately followed by `>` inside the arm block.
		depth := 0
		for j := bo + 1; j < bc; j++ {
			switch toks[j].Text {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			}
			if depth == 0 && toks[j].Text == "=" && j+1 < bc && toks[j+1].Text == ">" {
				ps := matchPatternStart(toks, j)
				spans = append(spans, span{open: ps - 1, close: j})
			}
		}
		mi = bc // arm blocks don't overlap; resume past this one.
	}
	return spans
}

// matchPatternStart returns the first token index of the arm pattern whose `=>` arrow's
// `=` is at eqIdx. Lifted from internal/pass/result.go patternStart (the shared qualified
// match locator): a `Qual.Variant(binding)` pattern walks back from its `)` to the
// qualifier; a bare `Qual.Variant` is three tokens; `_` is the rest arm.
func matchPatternStart(toks []scan.Token, eqIdx int) int {
	j := eqIdx - 1
	switch toks[j].Text {
	case ")":
		depth := 0
		k := j
		for ; k >= 0; k-- {
			switch toks[k].Text {
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
		return j - 2
	}
}

// missingFields returns the declared field names not present in the literal, in
// declared order.
func missingFields(declared []analyze.Field, present map[string]bool) []string {
	var missing []string
	for _, f := range declared {
		if !present[f.Name] {
			missing = append(missing, f.Name)
		}
	}
	return missing
}

// isBraceKeyword reports whether name is a keyword that legitimately precedes a "{"
// without being a struct-literal type name, so `name {` is not a composite literal.
func isBraceKeyword(name string) bool {
	switch name {
	case "struct", "interface", "func", "map", "switch", "select", "for",
		"if", "else", "range", "go", "defer", "return", "import", "package":
		return true
	}
	return false
}

// quoteJoin renders names as a comma-separated list of backtick-quoted identifiers in
// the order given.
func quoteJoin(names []string) string {
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = "`" + n + "`"
	}
	return strings.Join(quoted, ", ")
}

// plural returns "s" when n != 1, for "field"/"fields" agreement.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// pronoun returns "it"/"them" for the trailing "set … explicitly" clause.
func pronoun(n int) string {
	if n == 1 {
		return "it"
	}
	return "them"
}
