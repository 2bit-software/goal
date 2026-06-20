package check

import (
	"fmt"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// checkConvert enforces feature 12 (derive-convert): a `derive func g(s S) T` must be
// total — every field of the target T must be reachable, either field-by-field from S
// (same type, or via a registered `from func`, or via built-in slice recursion) or via
// an explicit `Field: expr` override / `Field: _` skip in the body. A target field with
// no resolvable source and no exception is an Error.
//
// Reuse, not reinvention:
//   - This walk mirrors the derive pass (internal/pass/derive.go): expandDerives finds
//     each `derive func`, reads its source param (name + type S), target type T, and
//     fallibility, then parseOverrides reads the body's `Field: expr` / `Field: _`
//     entries; genConversion / resolveField decide each `...derive` field's strategy.
//     This check re-derives the same facts and asserts resolvability instead of splicing.
//   - analyze.Tables.Structs gives both field lists (by type name); analyze.Tables
//     .FromRegistry resolves a field whose source and target types differ. This
//     generalizes the 06 From-totality check from error types to any type pair.
//   - Must run pre-lowering: the derive pass replaces the whole `derive func` with the
//     generated field-by-field function, erasing the construct this guarantee inspects.
//
// Defer-boundary (emit a located Warning, never a false incompleteness Error):
//   - unresolved-derive-type: the target T (or the source S) is not a `type … struct`
//     declared in this file (out-of-package type) — its field set is unreadable, so
//     completeness can't be proven for any field.
//   - unresolved-derive-field: a `...derive` target field whose resolution needs the
//     recursion classes the v1 deriver keeps minimal — map (`map[K]A`→`map[K]B`),
//     Option (`Option[A]`→`Option[B]`), `Option[T]`↔`*T`, and nested-struct
//     auto-recursion — or the two bespoke audit shapes (pmk_upgrade, patterns JSON).
//     The field may yet be total via a rule this lexical check doesn't model, so it is
//     deferred rather than flagged.
//
// No analyze.Tables extension was needed: Structs + FromRegistry carry every fact this
// guarantee reads.
func checkConvert(src string, t *analyze.Tables) ([]Diagnostic, error) {
	toks := scan.Lex(src)
	var diags []Diagnostic
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].Text != "derive" || toks[i+1].Text != "func" {
			continue
		}
		diags = append(diags, checkOneDerive(src, toks, i, t)...)
	}
	return diags, nil
}

// checkOneDerive verifies one `derive func` at token i (the `derive` keyword). It mirrors
// expandDerives's parse: name, source param (name + type S), return type → target T +
// fallibility, and the optional body's overrides. It then asserts that every unmentioned
// target field of T is resolvable from S, emitting an Error for an unsourced/unbridged
// field and a located Warning where resolution needs a deferred recursion class.
func checkOneDerive(src string, toks []scan.Token, i int, t *analyze.Tables) []Diagnostic {
	pos := toks[i].Start // located at the `derive` keyword
	open := indexOfTok(toks, i+2, "(")
	if open < 0 {
		return nil // malformed; the derive pass rejects it
	}
	closeP := scan.MatchParen(toks, open)
	if closeP < 0 || open+1 >= closeP {
		return nil
	}
	srcType := strings.TrimSpace(src[toks[open+1].End:toks[closeP].Start])

	// Return type runs to the body `{` or, if bodyless, to end of line (mirrors expandDerives).
	afterParams := toks[closeP].End
	brace := strings.IndexByte(src[afterParams:], '{')
	nl := strings.IndexByte(src[afterParams:], '\n')
	hasBody := brace >= 0 && (nl < 0 || brace < nl)

	var retType string
	overridden := map[string]bool{} // lowercased target field names named in the body
	if hasBody {
		bodyOpen := afterParams + brace
		retType = strings.TrimSpace(src[afterParams:bodyOpen])
		bodyOpenTok := tokenAtOffset(toks, bodyOpen)
		if bodyOpenTok < 0 {
			return nil
		}
		bodyClose := scan.MatchBrace(toks, bodyOpenTok)
		for _, o := range parseOverrides(src, toks, bodyOpenTok, bodyClose) {
			overridden[strings.ToLower(o.Name)] = true
		}
	} else {
		end := len(src)
		if nl >= 0 {
			end = afterParams + nl
		}
		retType = strings.TrimSpace(src[afterParams:end])
	}

	tgtType, fallible := splitReturn(retType)

	tgtFields, ok := t.Structs[tgtType]
	if !ok {
		return []Diagnostic{{
			Pos:      pos,
			Severity: Warning,
			Feature:  "12-derive-convert",
			Code:     "unresolved-derive-type",
			Message: fmt.Sprintf("cannot verify `derive func` is total: target type `%s` is not a struct declared in this file — completeness deferred",
				tgtType),
		}}
	}
	srcFields, srcKnown := t.Structs[srcType]
	if !srcKnown {
		return []Diagnostic{{
			Pos:      pos,
			Severity: Warning,
			Feature:  "12-derive-convert",
			Code:     "unresolved-derive-type",
			Message: fmt.Sprintf("cannot verify `derive func` is total: source type `%s` is not a struct declared in this file — completeness deferred",
				srcType),
		}}
	}

	var diags []Diagnostic
	for _, f := range tgtFields {
		if overridden[strings.ToLower(f.Name)] {
			continue // `Field: expr` override or `Field: _` skip — author-supplied, satisfied
		}
		sf, found := findField(srcFields, f.Name)
		if !found {
			diags = append(diags, Diagnostic{
				Pos:      pos,
				Severity: Error,
				Feature:  "12-derive-convert",
				Code:     "unsourced-field",
				Message: fmt.Sprintf("`derive func` target field `%s.%s` has no same-named source field on `%s` and no explicit override — add `%s: …` or `%s: _`, or a same-named field on `%s`",
					tgtType, f.Name, srcType, f.Name, f.Name, srcType),
			})
			continue
		}
		if d, ok := resolvableField(pos, tgtType, srcType, f, sf, fallible, t.FromRegistry); !ok {
			diags = append(diags, d)
		}
	}
	return diags
}

// resolvableField decides whether the `...derive` target field f (type TF) is reachable
// from the same-named source field sf (type SF), mirroring resolveField's strategy order:
// same type, registry conversion, or built-in slice recursion. It returns ok=true when
// resolvable. When not resolvable it returns a Diagnostic: an Error (unbridged-field) for
// a concrete unreachable pair, or a Warning (unresolved-derive-field) for a pair whose
// resolution needs a deferred recursion class (map / Option / nested struct).
func resolvableField(pos int, tgtType, srcType string, f, sf analyze.Field, fallible bool, reg map[[2]string]analyze.ConvEntry) (Diagnostic, bool) {
	tf := strings.TrimSpace(f.Type)
	sfType := strings.TrimSpace(sf.Type)
	if sfType == tf {
		return Diagnostic{}, true // direct assignment
	}
	if e, ok := reg[[2]string{sfType, tf}]; ok {
		if e.Fallible && !fallible {
			// A registered conversion exists but it is fallible while this derive is
			// total — a real, located violation (the deriver can't thread `?`). Mirrors
			// resolveField's "declare the derive func returning (T, error)" error.
			return Diagnostic{
				Pos:      pos,
				Severity: Error,
				Feature:  "12-derive-convert",
				Code:     "fallible-in-total-derive",
				Message: fmt.Sprintf("`derive func` target field `%s.%s` needs the fallible conversion `%s` (`%s`→`%s`), but this derive is total — declare it returning `(%s, error)`",
					tgtType, f.Name, e.Name, sfType, tf, tgtType),
			}, false
		}
		return Diagnostic{}, true // registry conversion (total, or fallible into a fallible derive)
	}
	// Built-in slice recursion: []A -> []B when A -> B resolves totally (v1 rule).
	if strings.HasPrefix(sfType, "[]") && strings.HasPrefix(tf, "[]") {
		a := strings.TrimSpace(sfType[2:])
		b := strings.TrimSpace(tf[2:])
		if a == b {
			return Diagnostic{}, true
		}
		if e, ok := reg[[2]string{a, b}]; ok && !e.Fallible {
			return Diagnostic{}, true
		}
		// Slice of a pair with no total element conversion: defer — the element may be
		// bridged by a recursion class this lexical check doesn't model.
		return deferDeriveField(pos, tgtType, srcType, f, sfType), false
	}
	// Map / Option / nested-struct recursion and the bespoke audit shapes: deferred per
	// the audit (v1 keeps these minimal) — never a false incompleteness Error.
	if isDeferredShape(sfType) || isDeferredShape(tf) {
		return deferDeriveField(pos, tgtType, srcType, f, sfType), false
	}
	// A concrete, named type pair with no registered conversion and no recursion rule:
	// a real unbridged field. This is the footgun the feature exists to kill.
	return Diagnostic{
		Pos:      pos,
		Severity: Error,
		Feature:  "12-derive-convert",
		Code:     "unbridged-field",
		Message: fmt.Sprintf("`derive func` target field `%s.%s` is `%s` but its source `%s.%s` is `%s`, and no `from func` converts `%s`→`%s` — register one or override the field explicitly",
			tgtType, f.Name, tf, srcType, sf.Name, sfType, sfType, tf),
	}, false
}

// isDeferredShape reports whether a type expression names one of the v1-deferred recursion
// classes — map, Option, or a pointer (`Option[T]↔*T` bridge) — whose resolution this
// lexical check does not model and therefore must not flag as incomplete.
func isDeferredShape(typ string) bool {
	typ = strings.TrimSpace(typ)
	return strings.HasPrefix(typ, "map[") ||
		strings.HasPrefix(typ, "Option[") ||
		strings.HasPrefix(typ, "*")
}

// deferDeriveField builds the located Warning for a `...derive` field whose resolution
// needs a deferred recursion class (map / Option / nested struct / slice-of-unbridged).
func deferDeriveField(pos int, tgtType, srcType string, f analyze.Field, sfType string) Diagnostic {
	return Diagnostic{
		Pos:      pos,
		Severity: Warning,
		Feature:  "12-derive-convert",
		Code:     "unresolved-derive-field",
		Message: fmt.Sprintf("cannot verify `derive func` field `%s.%s` (`%s`→`%s`): needs a map/Option/nested recursion the v1 checker keeps minimal — completeness deferred for this field",
			tgtType, f.Name, sfType, strings.TrimSpace(f.Type)),
	}
}

// The following are private ports of the derive pass's locating/parsing helpers
// (internal/pass/derive.go), kept identical so this check reads a `derive func` exactly
// the way the lowering does. They are not importable across packages, so they are mirrored
// here per the spine's reuse contract (re-lex, reuse the locator, assert instead of splice).

// parseOverrides reads `Field: expr` / `Field: _` entries from a derive body's returned
// composite literal (ignoring the `...derive(src)` element). Mirrors pass.parseOverrides.
func parseOverrides(src string, toks []scan.Token, openIdx, closeIdx int) []analyze.Field {
	ret := indexOfTok(toks, openIdx+1, "return")
	if ret < 0 {
		return nil
	}
	litOpen := indexOfTok(toks, ret+1, "{")
	if litOpen < 0 || litOpen >= closeIdx {
		return nil
	}
	litClose := scan.MatchBrace(toks, litOpen)
	var out []analyze.Field
	depth := 0
	for k := litOpen + 1; k < litClose; k++ {
		switch toks[k].Text {
		case "{", "[", "(":
			depth++
		case "}", "]", ")":
			depth--
		}
		if depth == 0 && scan.IsIdent(toks[k].Text) && k+1 < litClose && toks[k+1].Text == ":" {
			out = append(out, analyze.Field{Name: toks[k].Text})
		}
	}
	return out
}

// splitReturn splits a return type into target and fallibility: `(T, error)` -> (T, true);
// a bare `T` -> (T, false). Mirrors pass.splitReturn.
func splitReturn(ret string) (tgt string, fallible bool) {
	ret = strings.TrimSpace(ret)
	if strings.HasPrefix(ret, "(") && strings.HasSuffix(ret, ")") {
		first, _, _ := strings.Cut(ret[1:len(ret)-1], ",")
		return strings.TrimSpace(first), true
	}
	return ret, false
}

// findField returns the source field whose name matches (case-insensitively). Mirrors
// pass.findField.
func findField(fields []analyze.Field, name string) (analyze.Field, bool) {
	for _, f := range fields {
		if strings.EqualFold(f.Name, name) {
			return f, true
		}
	}
	return analyze.Field{}, false
}

// indexOfTok returns the index of the first token with text txt at or after from, or -1.
// Mirrors pass.indexOfTok.
func indexOfTok(toks []scan.Token, from int, txt string) int {
	for k := from; k < len(toks); k++ {
		if toks[k].Text == txt {
			return k
		}
	}
	return -1
}

// tokenAtOffset returns the index of the token starting at byte offset, or -1. Mirrors
// pass.tokenAtOffset.
func tokenAtOffset(toks []scan.Token, offset int) int {
	for k := range toks {
		if toks[k].Start == offset {
			return k
		}
	}
	return -1
}
