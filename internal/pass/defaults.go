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
			entries = append(entries, fmt.Sprintf("%s: %s", f.Name, zeroLit(f.Type, t.TypeDecls, 0)))
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

// zeroLit returns the explicit Go zero literal for a declared field type. Untyped
// constants (`0`, `""`, `false`) are assignable to defined types, so a defined type
// like `type Role int` correctly defaults to `0`. depth guards alias chains.
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
		if under, ok := decls[scan.BaseType(typ)]; ok {
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
	// zero. A named interface from another package would want `nil`; not recoverable
	// without a type system — out of scope.
	return typ + "{}"
}
