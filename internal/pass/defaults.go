package pass

import (
	"fmt"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// Defaults expands the explicit-defaults escape hatch `...defaults` (feature 08,
// §8.5). Required-field construction is a checker concern that is erased; the one
// rewrite here names the "I do want the zeros for the rest" case and lowers it to
// explicit per-field zero values so the generated literal is also complete:
//
//	User{name: n, ...defaults}  ->  User{name: n, email: "", active: false}
//
// Field zeros are recovered from the struct's declared field types (analyze.Structs)
// through alias chains (analyze.TypeDecls).
func Defaults(src string, t *analyze.Tables) (string, error) {
	toks := scan.Lex(src)
	var reps []scan.Replacement
	for i := range toks {
		if !isDefaultsForm(toks, i) {
			continue
		}
		// The form occupies tokens `.` `.` `.` `defaults`, i.e. toks[i-3..i].
		spanStart, spanEnd := toks[i-3].Start, toks[i].End

		openIdx := enclosingBrace(toks, i-4)
		if openIdx < 1 {
			return "", fmt.Errorf("`...defaults` at offset %d is not inside a struct literal", spanStart)
		}
		typeName := toks[openIdx-1].Text
		fields, ok := t.Structs[typeName]
		if !ok {
			return "", fmt.Errorf("`...defaults` for unknown struct type %q (no `type %s struct{…}` in this file)", typeName, typeName)
		}

		present := presentFields(toks, openIdx)
		var entries []string
		for _, f := range fields {
			if present[f.Name] {
				continue
			}
			// `...defaults` only fills fields whose zero is safe. A field whose zero is
			// a latent hazard (nil map/pointer/chan/func, a sum type with no valid
			// variant, a nil method-interface) is a located compile error, not a silent
			// zero — set it explicitly, or use Option[T] for an optional reference.
			if reason := zeroSafety(f.Type, t, 0); reason != "" {
				line, col := lineCol(src, spanStart)
				return "", fmt.Errorf("`...defaults` at %d:%d cannot default field `%s` of type `%s`: %s",
					line, col, f.Name, f.Type, reason)
			}
			entries = append(entries, fmt.Sprintf("%s: %s", f.Name, analyze.ZeroLit(f.Type, t.TypeDecls, 0)))
		}
		reps = append(reps, scan.Replacement{Start: spanStart, End: spanEnd, Text: strings.Join(entries, ", ")})
	}
	return scan.Splice(src, 0, len(src), reps), nil
}

// isDefaultsForm reports whether toks[i] is the `defaults` ident of a `...defaults`
// element (three leading `.` tokens).
func isDefaultsForm(toks []scan.Token, i int) bool {
	return i >= 3 && toks[i].Text == "defaults" &&
		toks[i-1].Text == "." && toks[i-2].Text == "." && toks[i-3].Text == "."
}

// enclosingBrace scans backward from `from` to find the "{" of the composite literal
// that directly contains it (depth-aware). Returns -1 if none.
func enclosingBrace(toks []scan.Token, from int) int {
	depth := 0
	for k := from; k >= 0; k-- {
		switch toks[k].Text {
		case "}", "]", ")":
			depth++
		case "{", "[", "(":
			if depth == 0 {
				if toks[k].Text == "{" {
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
func presentFields(toks []scan.Token, openIdx int) map[string]bool {
	present := map[string]bool{}
	closeIdx := scan.MatchBrace(toks, openIdx)
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

// zeroSafety reports why a field of type typ has no safe zero to fill via `...defaults`,
// or "" when its zero is safe. The traversal mirrors zeroLit: a type whose zero is a
// usable value (primitive, struct, array, nil slice, `error`, bare interface) is safe;
// a type whose nil zero panics or deadlocks on normal use (pointer, map, chan, func, a
// method-bearing named interface) or a sum type that has no valid zero variant
// (`enum` / sealed interface) is rejected. depth guards alias chains.
func zeroSafety(typ string, t *analyze.Tables, depth int) string {
	typ = strings.TrimSpace(typ)
	switch {
	case strings.HasPrefix(typ, "*"):
		return "a nil pointer has no safe zero — set it explicitly, or use Option[T] for an optional value"
	case strings.HasPrefix(typ, "map["):
		return "a nil map panics on write — set it explicitly (e.g. `" + typ + "{}`)"
	case strings.HasPrefix(typ, "chan"):
		return "a nil channel blocks forever — set it explicitly"
	case strings.HasPrefix(typ, "func"):
		return "a nil func panics when called — set it explicitly"
	case strings.HasPrefix(typ, "interface"):
		// Bare `interface{}` has no methods, so its nil is harmless; a method-bearing
		// interface literal panics on a nil method call.
		if strings.TrimSpace(typ[len("interface"):]) == "{}" {
			return ""
		}
		return "a nil interface has no safe zero — set it explicitly"
	case typ == "any", typ == "error":
		return "" // bare any: no methods; nil error is the success value
	case strings.HasPrefix(typ, "[]"):
		return "" // a nil slice is safe: range/len/append all work on it
	case strings.HasPrefix(typ, "["):
		return "" // array: composite zero is a usable value
	}
	switch typ {
	case "string", "bool",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"byte", "rune", "float32", "float64", "complex64", "complex128":
		return ""
	}
	base := scan.BaseType(typ)
	if _, ok := t.Enums[base]; ok || t.Sealed[base] {
		return "a sum type has no valid zero variant — set it explicitly"
	}
	if depth < 8 {
		if under, ok := t.TypeDecls[base]; ok {
			switch under {
			case "struct":
				return ""
			case "interface":
				return "a nil interface has no safe zero — set it explicitly"
			default:
				return zeroSafety(under, t, depth+1)
			}
		}
	}
	// Unknown external named type: assume struct-like (as zeroLit does) — treat as safe.
	return ""
}

// lineCol converts a byte offset into 1-based line and column numbers for a located
// diagnostic.
func lineCol(src string, off int) (line, col int) {
	if off > len(src) {
		off = len(src)
	}
	line = 1 + strings.Count(src[:off], "\n")
	col = off - (strings.LastIndexByte(src[:off], '\n') + 1) + 1
	return line, col
}
