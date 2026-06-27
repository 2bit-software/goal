package backend

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"goal/internal/ast"
	"goal/internal/sema"
)

// lower.go holds the goal-construct encoders the emitter (emit.go) splices into
// the plain-Go output: the §8.1 closed-sum encoding for enums and sealed
// interfaces, and the helpers the §8.5 `implements` lowering needs. These mirror
// the known-good encoding the legacy splice passes produce (internal/pass/
// enums.go, internal/pass/implements.go), reimplemented over the resolved AST
// facts in sema.Info so the new (AST) backend emits the same shapes.
//
// The encoders emit syntactically valid, token-correct Go text (the emitter's
// format-once discipline) — readability is the Formatter's job.

// fileIdentSet collects every identifier name that appears in f. It seeds the
// emitter's scope-aware gensym (emit.go) so a generated name — the §8.3/§8.4
// success/error returns, the `?`/match temporaries — never shadows or collides
// with a name the source already uses. This is the structural replacement for the
// fixed `__goal_` prefix the splice engine and the US-034 seed lowering used
// (US-035): instead of a reserved magic prefix that merely *assumes* it never
// clashes, names are minted against the actual identifiers in scope.
func fileIdentSet(f *ast.File) map[string]bool {
	set := map[string]bool{}
	if f == nil {
		return set
	}
	ast.Walk(identFinder(func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			set[id.Name] = true
		}
		return true
	}), f)
	return set
}

// roKind classifies a function result type as one of goal's lowered core types.
type roKind int

const (
	roNone         roKind = iota // not a Result/Option result
	roResultOpen                 // open-E Result[T, error] -> native (T, error)
	roOption                     // Option[T] -> *T
	roResultClosed               // closed-E Result[T, E] (E != error) -> Ok[T,E]/Err[T,E] sum
)

// resultPrelude is the generic sum encoding (spec §8.1) a closed-E Result program
// needs in scope: the marker interface plus the Ok/Err carriers and their marker
// methods. The emitter injects it once per file (see needsResultPrelude). It
// mirrors internal/pass.ResultPreamble verbatim (a known-good, build+vet-clean
// encoding); the format-once Formatter normalizes its layout.
const resultPrelude = `type Result[T, E any] interface{ isResult() }
type Ok[T, E any] struct{ Value T }
type Err[T, E any] struct{ Value E }

func (Ok[T, E]) isResult()  {}
func (Err[T, E]) isResult() {}`

// needsResultPrelude reports whether any function in info returns a closed-E
// Result, i.e. whether the file needs resultPrelude in scope. Nil-safe. Mirrors
// internal/pass.NeedsResultPrelude but reads the sema signatures.
func needsResultPrelude(info *sema.Info) bool {
	if info == nil {
		return false
	}
	for _, sig := range info.FuncSignatures {
		if sig.Mode == sema.ModeResultClosed {
			return true
		}
	}
	return false
}

// resultOptionKind classifies a function's single unnamed result type as open-E
// Result, Option, or neither, and returns the success type expression (the T in
// Result[T, error] / Option[T]) for the two recognized cases. A closed-E Result
// (Result[T, E] where E is not error) is roNone here — its sum encoding is a
// later story (US-037). A named or multi-value result is roNone.
func resultOptionKind(t *ast.FuncType) (roKind, ast.Expr) {
	if t == nil || t.Results == nil || len(t.Results.List) != 1 {
		return roNone, nil
	}
	f := t.Results.List[0]
	if len(f.Names) != 0 {
		return roNone, nil
	}
	switch ty := f.Type.(type) {
	case *ast.IndexListExpr:
		if id, ok := ty.X.(*ast.Ident); ok && id.Name == "Result" &&
			len(ty.Indices) == 2 && isErrorIdent(ty.Indices[1]) {
			return roResultOpen, ty.Indices[0]
		}
	case *ast.IndexExpr:
		if id, ok := ty.X.(*ast.Ident); ok && id.Name == "Option" {
			return roOption, ty.Index
		}
	}
	return roNone, nil
}

// isErrorIdent reports whether x is the bare type name `error`.
func isErrorIdent(x ast.Expr) bool {
	id, ok := x.(*ast.Ident)
	return ok && id.Name == "error"
}

// matchQualifier returns the enum/type qualifier of a match's first
// variant-pattern arm (`Result`, `Option`, or an enum name), or "" when the first
// arm is not a qualified variant pattern. It picks the lowering strategy for a
// statement-position match the same way the splice engine's scan.MatchQualifier
// does, but reads it off the parsed arm instead of the token stream.
func matchQualifier(m *ast.MatchExpr) string {
	for _, arm := range m.Arms {
		vp, ok := arm.Pattern.(*ast.VariantPattern)
		if !ok {
			continue
		}
		if id, ok := vp.Enum.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

// usesIdent reports whether the subtree rooted at n references an identifier
// named name. It is used to decide whether a match arm captures its payload
// binding (so an unused Ok value is discarded with `_` rather than declared and
// left unused, which would fail to compile).
func usesIdent(n ast.Node, name string) bool {
	found := false
	ast.Walk(identFinder(func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok && id.Name == name {
			found = true
		}
		return !found // stop descending once found
	}), n)
	return found
}

// identFinder adapts a func to the ast.Visitor interface; it keeps descending
// while the func returns true.
type identFinder func(ast.Node) bool

func (f identFinder) Visit(n ast.Node) ast.Visitor {
	if n == nil || !f(n) {
		return nil
	}
	return f
}

// enumOf returns the resolved enum named name, or nil when info or the enum is
// absent (nil-safe so the plain-Go path, which carries no enums, is harmless).
func enumOf(info *sema.Info, name string) *sema.Enum {
	if info == nil || info.Enums == nil {
		return nil
	}
	return info.Enums[name]
}

// isSealed reports whether name is a sealed interface (nil-safe).
func isSealed(info *sema.Info, name string) bool {
	return info != nil && info.Sealed != nil && info.Sealed[name]
}

// genEnum emits the §8.1 encoding for an enum: a marker interface, one struct per
// variant, and a marker method per variant. It mirrors internal/pass.genEnum but
// reads its variants from the resolved sema.Enum.
func genEnum(e *sema.Enum) string {
	marker := "is" + e.Name
	var b strings.Builder
	fmt.Fprintf(&b, "type %s interface{ %s() }\n\n", e.Name, marker)
	for _, v := range e.Variants {
		if len(v.Fields) == 0 {
			fmt.Fprintf(&b, "type %s_%s struct{}\n", e.Name, v.Name)
			continue
		}
		fmt.Fprintf(&b, "type %s_%s struct {\n", e.Name, v.Name)
		for _, f := range v.Fields {
			fmt.Fprintf(&b, "\t%s %s\n", exported(f.Name), f.Type)
		}
		b.WriteString("}\n")
	}
	b.WriteString("\n")
	for _, v := range e.Variants {
		fmt.Fprintf(&b, "func (%s_%s) %s() {}\n", e.Name, v.Name, marker)
	}
	return b.String()
}

// genSealedInterface emits a sealed interface as a marker interface
// (`type Name interface{ isName() }`). Mirrors internal/pass.genInterface.
func genSealedInterface(name string) string {
	return fmt.Sprintf("type %s interface{ is%s() }", name, name)
}

// genMarkerMethod emits the unexported marker method that admits typ into the
// sealed interface iface (`func (T) isI() {}`). Mirrors internal/pass.genMarker.
func genMarkerMethod(typ, iface string) string {
	return fmt.Sprintf("func (%s) is%s() {}", typ, iface)
}

// exported capitalizes the first rune so a goal field/label maps to an exported
// Go field name (`since` -> `Since`). Mirrors internal/pass.exported.
func exported(name string) string {
	if name == "" {
		return name
	}
	r := []rune(name)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// baseType strips a leading "*" and any "pkg." qualifier, yielding the bare type
// name (used to look up a local type). Mirrors scan.BaseType so lower.go need not
// import internal/scan.
func baseType(t string) string {
	t = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(t), "*"))
	if i := strings.LastIndexByte(t, '.'); i >= 0 {
		t = t[i+1:]
	}
	return t
}

// zeroLit returns the explicit Go zero literal for a declared type (§8.5). Mirrors
// analyze.ZeroLit: an untyped constant (`0`, `""`, `false`, `nil`) assignable to the
// field's defined type. decls maps a named type to its underlying form ("struct",
// "interface", or a type expression) so the zero is recoverable through alias
// chains; depth guards those chains. The `...defaults` expansion uses it.
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
	// Unknown named type with no local declaration: assume a struct-like composite zero.
	return typ + "{}"
}

// zeroSafety reports why a field of type typ has no safe zero to fill via
// `...defaults`, or "" when its zero is safe. Mirrors internal/pass.zeroSafety, but
// reads sum-type membership off sema.Info and alias chains off decls. A type whose
// nil zero panics/deadlocks on normal use (pointer, map, chan, func, method-bearing
// interface) or a sum type with no valid zero variant (enum / sealed interface) is
// rejected; a primitive, struct, array, nil slice, error, or bare interface is safe.
// depth guards alias chains.
func zeroSafety(typ string, decls map[string]string, info *sema.Info, depth int) string {
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
	base := baseType(typ)
	if enumOf(info, base) != nil || isSealed(info, base) {
		return "a sum type has no valid zero variant — set it explicitly"
	}
	if depth < 8 {
		if under, ok := decls[base]; ok {
			switch under {
			case "struct":
				return ""
			case "interface":
				return "a nil interface has no safe zero — set it explicitly"
			default:
				return zeroSafety(under, decls, info, depth+1)
			}
		}
	}
	// Unknown external named type: assume struct-like (as zeroLit does) — treat as safe.
	return ""
}

// needsFmtImport reports whether f contains a printf-message `assert` (one whose
// lowering emits `fmt.Sprintf`), so the emitter can inject `import "fmt"`.
func needsFmtImport(f *ast.File) bool {
	need := false
	ast.Walk(identFinder(func(n ast.Node) bool {
		if a, ok := n.(*ast.AssertStmt); ok && a.Msg != nil {
			need = true
		}
		return !need // stop walking once found
	}), f)
	return need
}

// importsPkg reports whether f already imports the package at the given path (e.g.
// "fmt"), so a duplicate import is not injected.
func importsPkg(f *ast.File, path string) bool {
	if f == nil {
		return false
	}
	quoted := strconv.Quote(path)
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok.String() != "import" {
			continue
		}
		for _, s := range gd.Specs {
			if is, ok := s.(*ast.ImportSpec); ok && is.Path != nil && is.Path.Value == quoted {
				return true
			}
		}
	}
	return false
}

// presentFieldNames returns the set of keyed field names already set in a struct
// composite literal's element list (the `name:` keys), so `...defaults` fills only
// the omitted fields.
func presentFieldNames(elts []ast.Expr) map[string]bool {
	present := map[string]bool{}
	for _, el := range elts {
		kv, ok := el.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if id, ok := kv.Key.(*ast.Ident); ok {
			present[id.Name] = true
		}
	}
	return present
}

// structFieldsOf returns the ordered fields of the named struct, nil-safe.
func structFieldsOf(info *sema.Info, name string) ([]sema.Field, bool) {
	if info == nil || info.Structs == nil {
		return nil, false
	}
	fs, ok := info.Structs[name]
	return fs, ok
}

// pointerReceiverSet returns the set of type names that have at least one
// pointer-receiver method (`func (x *T) ...`). The §8.5 implements assertion must
// address such a type as `(*T)(nil)` rather than `T{}`, since the value type does
// not carry the pointer-receiver method. sema.Info.Methods is keyed by the
// star-stripped receiver name and so cannot answer this, so the set is computed
// here by walking the file's declarations. Mirrors
// internal/pass.scanPointerReceivers.
func pointerReceiverSet(f *ast.File) map[string]bool {
	set := map[string]bool{}
	if f == nil {
		return set
	}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv == nil || len(fd.Recv.List) == 0 {
			continue
		}
		if star, ok := fd.Recv.List[0].Type.(*ast.StarExpr); ok {
			if id, ok := star.X.(*ast.Ident); ok {
				set[id.Name] = true
			}
		}
	}
	return set
}
