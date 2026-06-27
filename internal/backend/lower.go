package backend

import (
	"fmt"
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

// Gensym names for the open-E Result / Option lowering (§8.3, §8.4). They mirror
// internal/pass/pass.go so the AST backend emits the same shapes the splice
// engine does (US-035 retires the literal __goal_ prefix for `?` propagation;
// US-042 regenerates the exact goldens). The behavioral tier judges by build +
// vet, so the exact spelling is not load-bearing — only that the names a lowered
// signature declares match the names its body references.
const (
	okName   = "__goal_ok"   // named success return / Ok arm-binding target
	errName  = "__goal_err"  // named error return / Err arm-binding target
	valName  = "__goal_v"    // Ok value captured at a statement-position match
	someName = "__goal_some" // boxed Some value when the payload is not addressable
	optBase  = "__goal_o"    // Option pointer temporary at a statement-position match
)

// roKind classifies a function result type as one of goal's lowered core types.
type roKind int

const (
	roNone       roKind = iota // not a Result/Option result
	roResultOpen               // open-E Result[T, error] -> native (T, error)
	roOption                   // Option[T] -> *T
)

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
