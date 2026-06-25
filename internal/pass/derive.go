package pass

import (
	"fmt"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// Derive implements feature 12: type-directed, completeness-checked struct
// conversion. It owns every `from func` leaf — stripping the `from` modifier so the
// leaf becomes a plain Go function (the registry was built by analyze, shared with
// the closed-E pass) — and expands each `derive func` into field-by-field Go,
// resolving every target field through the registry.
//
// `...derive(src)` fills the unmentioned fields (the parallel of `...defaults`);
// `Field: expr` is a verbatim override; `Field: _` skips. Per the no-checker
// discipline, an unresolvable field is DEFERRED with a located error, never silently
// zeroed — the footgun this feature exists to kill. Slice-container recursion is
// implemented; map/Option/nested recursion follow the same rule (minimal in v1).
func Derive(src string, t *analyze.Tables) (string, error) {
	toks := scan.Lex(src)
	var reps []scan.Replacement

	// Strip `from ` from every leaf conversion (feature 06 error leaves and feature
	// 12 any-type leaves alike).
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Text == "from" && toks[i+1].Text == "func" {
			reps = append(reps, scan.Replacement{Start: toks[i].Start, End: toks[i+1].Start, Text: ""})
		}
	}

	deriveReps, err := expandDerives(src, toks, t)
	if err != nil {
		return "", err
	}
	reps = append(reps, deriveReps...)
	return scan.Splice(src, 0, len(src), reps), nil
}

// expandDerives replaces each `derive func` declaration with a generated function.
func expandDerives(src string, toks []scan.Token, t *analyze.Tables) ([]scan.Replacement, error) {
	var reps []scan.Replacement
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].Text != "derive" || toks[i+1].Text != "func" {
			continue
		}
		name := toks[i+2].Text
		open := indexOfTok(toks, i+2, "(")
		closeP := scan.MatchParen(toks, open)
		srcName := toks[open+1].Text
		srcType := strings.TrimSpace(src[toks[open+1].End:toks[closeP].Start])

		// Return type runs to the body `{` or, if bodyless, to end of line.
		afterParams := toks[closeP].End
		brace := strings.IndexByte(src[afterParams:], '{')
		nl := strings.IndexByte(src[afterParams:], '\n')
		hasBody := brace >= 0 && (nl < 0 || brace < nl)

		var retType string
		var overrides []analyze.Field
		var declEnd int
		if hasBody {
			bodyOpen := afterParams + brace
			retType = strings.TrimSpace(src[afterParams:bodyOpen])
			bodyOpenTok := tokenAtOffset(toks, bodyOpen)
			bodyClose := scan.MatchBrace(toks, bodyOpenTok)
			declEnd = toks[bodyClose].End
			overrides = parseOverrides(src, toks, bodyOpenTok, bodyClose)
		} else {
			end := len(src)
			if nl >= 0 {
				end = afterParams + nl
			}
			retType = strings.TrimSpace(src[afterParams:end])
			declEnd = end
		}

		tgtType, fallible := splitReturn(retType)
		gen, err := genConversion(name, srcName, srcType, tgtType, retType, fallible, overrides, t)
		if err != nil {
			return nil, err
		}
		reps = append(reps, scan.Replacement{Start: toks[i].Start, End: declEnd, Text: gen})
	}
	return reps, nil
}

// parseOverrides reads `Field: expr` / `Field: _` entries from a derive body's
// returned composite literal (ignoring the `...derive(src)` element).
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
			name := toks[k].Text
			valStart := toks[k+2].Start
			valEnd := topLevelCommaOrClose(toks, k+2, litClose)
			out = append(out, analyze.Field{Name: name, Type: strings.TrimSpace(src[valStart:valEnd])})
		}
	}
	return out
}

// genConversion produces the Go function body for one derived conversion.
func genConversion(name, srcName, srcType, tgtType, retType string, fallible bool, overrides []analyze.Field, t *analyze.Tables) (string, error) {
	tgtFields, ok := t.Structs[tgtType]
	if !ok {
		return "", fmt.Errorf("derive %s: unknown target struct %q", name, tgtType)
	}
	srcFields := t.Structs[srcType]

	overridden := map[string]bool{}
	for _, o := range overrides {
		overridden[strings.ToLower(o.Name)] = true
	}

	var b strings.Builder
	fmt.Fprintf(&b, "func %s(%s %s) %s {\n", name, srcName, srcType, retType)
	b.WriteString("var out " + tgtType + "\n")

	for _, o := range overrides { // explicit overrides first, `_` => leave zero
		if strings.TrimSpace(o.Type) == "_" {
			continue
		}
		fmt.Fprintf(&b, "out.%s = %s\n", o.Name, o.Type)
	}

	tempN := 0
	for _, f := range tgtFields { // `...derive(src)`: remaining fields, registry-resolved
		if overridden[strings.ToLower(f.Name)] {
			continue
		}
		sf, found := findField(srcFields, f.Name)
		if !found {
			return "", fmt.Errorf("derive %s: target field %q of %s is not sourced from %s (add an explicit `%s: …` or a `from func`)", name, f.Name, tgtType, srcType, f.Name)
		}
		stmts, err := resolveField("out."+f.Name, srcName+"."+sf.Name, sf.Type, f.Type, t, fallible, &tempN)
		if err != nil {
			return "", fmt.Errorf("derive %s, field %q: %w", name, f.Name, err)
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

// resolveField emits the statements assigning a converted source field to a target
// field, choosing the strategy by (source type -> target type).
func resolveField(dst, srcExpr, sf, tf string, t *analyze.Tables, fallibleOK bool, tempN *int) ([]string, error) {
	reg := t.FromRegistry
	sf, tf = strings.TrimSpace(sf), strings.TrimSpace(tf)
	if sf == tf {
		return []string{fmt.Sprintf("%s = %s", dst, srcExpr)}, nil
	}
	if e, ok := reg[[2]string{sf, tf}]; ok {
		if !e.Fallible {
			return []string{fmt.Sprintf("%s = %s(%s)", dst, e.Name, srcExpr)}, nil
		}
		if !fallibleOK {
			return nil, fmt.Errorf("conversion %s->%s is fallible; declare the derive func returning (T, error)", sf, tf)
		}
		v := fmt.Sprintf("__goal_v%d", *tempN)
		*tempN++
		return []string{
			fmt.Sprintf("%s, err := %s(%s)", v, e.Name, srcExpr),
			"if err != nil {\nreturn out, err\n}",
			fmt.Sprintf("%s = %s", dst, v),
		}, nil
	}
	// Pointer / Option recursion: *A -> *B (and Option[A] -> Option[B], which lowers to
	// the same pointer strategy) when A -> B resolves total. A nil source stays the zero
	// (nil) of the target pointer; a non-nil one is converted and re-addressed.
	if si, ok := ptrInner(sf); ok {
		ti, ok := ptrInner(tf)
		if !ok {
			return nil, fmt.Errorf("no conversion %s -> %s in scope", sf, tf)
		}
		elem, err := elemConv(si, ti, reg)
		if err != nil {
			return nil, err
		}
		v := fmt.Sprintf("__goal_p%d", *tempN)
		*tempN++
		return []string{
			fmt.Sprintf("if %s != nil {\n%s := %s\n%s = &%s\n}", srcExpr, v, elem("*"+srcExpr), dst, v),
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
	// Fixed-array recursion: [N]A -> [N]B (same length) when A -> B resolves total. The
	// target array is already zero; convert element-by-element in place.
	if sn, se, ok := arrElem(sf); ok {
		tn, te, ok := arrElem(tf)
		if !ok || sn != tn {
			return nil, fmt.Errorf("no conversion %s -> %s in scope", sf, tf)
		}
		elem, err := elemConv(se, te, reg)
		if err != nil {
			return nil, err
		}
		return []string{
			fmt.Sprintf("for i := range %s {\n%s = %s\n}", srcExpr, dst+"[i]", elem(srcExpr+"[i]")),
		}, nil
	}
	// Map recursion: map[K]A -> map[K]B (same key type) when A -> B resolves total.
	if sk, sv, ok := mapKV(sf); ok {
		tk, tv, ok := mapKV(tf)
		if !ok || sk != tk {
			return nil, fmt.Errorf("no conversion %s -> %s in scope", sf, tf)
		}
		elem, err := elemConv(sv, tv, reg)
		if err != nil {
			return nil, err
		}
		return []string{
			fmt.Sprintf("%s = make(%s, len(%s))", dst, tf, srcExpr),
			fmt.Sprintf("for k, v := range %s {\n%s[k] = %s\n}", srcExpr, dst, elem("v")),
		}, nil
	}
	// Nested in-package struct recursion: A -> B when both are structs declared in this
	// package. Build the target in a temp field-by-field (each field resolved by the same
	// strategy order), then assign it — so the recursion composes with the outer expansion.
	// A registered `from func A->B`, checked above, takes priority over auto-recursion.
	if _, srcStruct := t.Structs[sf]; srcStruct {
		if _, tgtStruct := t.Structs[tf]; tgtStruct {
			v := fmt.Sprintf("__goal_s%d", *tempN)
			*tempN++
			stmts := []string{fmt.Sprintf("var %s %s", v, tf)}
			body, err := deriveBody(v, srcExpr, sf, tf, t, fallibleOK, tempN)
			if err != nil {
				return nil, err
			}
			stmts = append(stmts, body...)
			return append(stmts, fmt.Sprintf("%s = %s", dst, v)), nil
		}
	}
	return nil, fmt.Errorf("no conversion %s -> %s in scope", sf, tf)
}

// deriveBody emits the field-by-field assignments converting srcType to tgtType into the
// already-declared dstVar. It is the recursion core for a nested struct field: every
// target field must be sourced (same name) and resolvable from the source, or the whole
// conversion is refused with a located error (no field is silently zeroed).
func deriveBody(dstVar, srcExpr, srcType, tgtType string, t *analyze.Tables, fallible bool, tempN *int) ([]string, error) {
	tgtFields, ok := t.Structs[tgtType]
	if !ok {
		return nil, fmt.Errorf("unknown target struct %q", tgtType)
	}
	srcFields := t.Structs[srcType]
	var stmts []string
	for _, f := range tgtFields {
		sf, found := findField(srcFields, f.Name)
		if !found {
			return nil, fmt.Errorf("nested field %q of %s is not sourced from %s", f.Name, tgtType, srcType)
		}
		s, err := resolveField(dstVar+"."+f.Name, srcExpr+"."+sf.Name, sf.Type, f.Type, t, fallible, tempN)
		if err != nil {
			return nil, fmt.Errorf("nested field %q: %w", f.Name, err)
		}
		stmts = append(stmts, s...)
	}
	return stmts, nil
}

// ptrInner returns the pointee type of a pointer-strategy field — a `*A`, or the
// `Option[A]` that lowers to it — and whether s is one.
func ptrInner(s string) (string, bool) {
	if strings.HasPrefix(s, "*") {
		return strings.TrimSpace(s[1:]), true
	}
	if strings.HasPrefix(s, "Option[") && strings.HasSuffix(s, "]") {
		return strings.TrimSpace(s[len("Option[") : len(s)-1]), true
	}
	return "", false
}

// arrElem splits a fixed-size array type `[N]E` into its length text and element type,
// rejecting slices (`[]E`). The length is compared as text, so `[3]` matches `[3]` and a
// named constant matches the same name.
func arrElem(s string) (n, elem string, ok bool) {
	if !strings.HasPrefix(s, "[") || strings.HasPrefix(s, "[]") {
		return "", "", false
	}
	close := strings.IndexByte(s, ']')
	if close < 0 {
		return "", "", false
	}
	n = strings.TrimSpace(s[1:close])
	if n == "" {
		return "", "", false
	}
	return n, strings.TrimSpace(s[close+1:]), true
}

// mapKV splits a `map[K]V` type into key and value, honoring bracket nesting in the key
// (e.g. `map[[2]int]V`).
func mapKV(s string) (k, v string, ok bool) {
	if !strings.HasPrefix(s, "map[") {
		return "", "", false
	}
	depth := 0
	for i := len("map[") - 1; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return strings.TrimSpace(s[len("map["):i]), strings.TrimSpace(s[i+1:]), true
			}
		}
	}
	return "", "", false
}

// elemConv returns a renderer for the conversion of a single slice element from type
// a to type b (total conversions only in v1).
func elemConv(a, b string, reg map[[2]string]analyze.ConvEntry) (func(string) string, error) {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == b {
		return func(x string) string { return x }, nil
	}
	if e, ok := reg[[2]string{a, b}]; ok && !e.Fallible {
		return func(x string) string { return e.Name + "(" + x + ")" }, nil
	}
	return nil, fmt.Errorf("no total element conversion %s -> %s for slice recursion", a, b)
}

// splitReturn splits a return type into target and fallibility: `(T, error)` ->
// (T, true); a bare `T` -> (T, false).
func splitReturn(ret string) (tgt string, fallible bool) {
	ret = strings.TrimSpace(ret)
	if strings.HasPrefix(ret, "(") && strings.HasSuffix(ret, ")") {
		first, _, _ := strings.Cut(ret[1:len(ret)-1], ",")
		return strings.TrimSpace(first), true
	}
	return ret, false
}

// findField returns the source field whose name matches (case-insensitively).
func findField(fields []analyze.Field, name string) (analyze.Field, bool) {
	for _, f := range fields {
		if strings.EqualFold(f.Name, name) {
			return f, true
		}
	}
	return analyze.Field{}, false
}

// indexOfTok returns the index of the first token with text txt at or after from.
func indexOfTok(toks []scan.Token, from int, txt string) int {
	for k := from; k < len(toks); k++ {
		if toks[k].Text == txt {
			return k
		}
	}
	return -1
}

// tokenAtOffset returns the index of the token starting at byte offset, or -1.
func tokenAtOffset(toks []scan.Token, offset int) int {
	for k := range toks {
		if toks[k].Start == offset {
			return k
		}
	}
	return -1
}

// topLevelCommaOrClose returns the byte offset of the first top-level `,` (or the
// closing brace at closeIdx) bounding a composite-literal element value.
func topLevelCommaOrClose(toks []scan.Token, from, closeIdx int) int {
	depth := 0
	for k := from; k < closeIdx; k++ {
		switch toks[k].Text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		case ",":
			if depth == 0 {
				return toks[k].Start
			}
		}
	}
	return toks[closeIdx].Start
}
