package analyze

import (
	"strings"

	"goal/internal/scan"
)

// FuncSpan pairs a function body's current byte span with its analyzed signature, so a
// caller that has lost the original return type from lowered (or plain-Go) source can
// recover the enclosing function's mode and T/E types by re-scanning and looking up the
// name. The lowering passes and the `goal fix` rewriter both consult it, so it lives here
// in the name-keyed analysis package rather than in any one pass.
type FuncSpan struct {
	Lo, Hi int
	Sig    FuncSig
}

// FuncSpans returns one span per function in the current source, carrying its analyzed
// signature from the name-keyed tables (functions without a recorded signature are
// omitted).
func FuncSpans(toks []scan.Token, t *Tables) []FuncSpan {
	var spans []FuncSpan
	for _, f := range scan.ScanFuncs(toks) {
		if sig, ok := t.FuncSignatures[f.Name]; ok {
			spans = append(spans, FuncSpan{Lo: toks[f.BodyOpen].Start, Hi: toks[f.BodyClose].End, Sig: sig})
		}
	}
	return spans
}

// SigAt returns the signature of the function whose body contains byte offset off.
func SigAt(spans []FuncSpan, off int) (FuncSig, bool) {
	for _, s := range spans {
		if off >= s.Lo && off < s.Hi {
			return s.Sig, true
		}
	}
	return FuncSig{}, false
}

// ZeroLit returns the explicit Go zero literal for a declared type. Untyped constants
// (`0`, `""`, `false`) are assignable to defined types, so a defined type like
// `type Role int` correctly defaults to `0`. decls maps a named type to its underlying
// form ("struct", "interface", or a type expression) so the zero can be recovered through
// alias chains; depth guards those chains. The `...defaults` expansion and the `goal fix`
// bare-propagation matcher share this so they agree on what a type's zero looks like.
func ZeroLit(typ string, decls map[string]string, depth int) string {
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
				return ZeroLit(under, decls, depth+1)
			}
		}
	}
	// Unknown named type with no local declaration: assume a struct-like composite zero.
	// A named interface from another package would want `nil`; not recoverable without a
	// type system — out of scope.
	return typ + "{}"
}
