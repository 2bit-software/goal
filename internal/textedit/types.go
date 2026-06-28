package textedit

import "strings"

// BaseType strips a leading "*" and any "pkg." qualifier, yielding the bare type
// name (used to look up a local type or receiver type).
func BaseType(t string) string {
	t = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(t), "*"))
	if i := strings.LastIndexByte(t, '.'); i >= 0 {
		t = t[i+1:]
	}
	return t
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
		if under, ok := decls[BaseType(typ)]; ok {
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
