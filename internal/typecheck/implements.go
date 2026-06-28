package typecheck

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"

	"goal/internal/scan"
	"goal/internal/sema"
	"goal/internal/textedit"
)

// CheckImplements verifies, with real go/types identity, that every
// `type T struct implements I` clause holds. This is the depth version of feature 07:
// where the lexical check compares normalized signature *text* and defers any qualified
// or out-of-package interface, this resolves the interface through the type system, so
//   - an alias-equal-but-differently-spelled signature is no longer a false mismatch
//     (the documented §07 lexical ceiling), and
//   - a qualified interface like io.Writer is actually checked, not deferred.
//
// It locates each clause in the goal source (the clause is erased by lowering) and
// verifies it against the type-checked package, reporting at the clause position.
func CheckImplements(p *Package) []Diagnostic {
	if p.Types == nil {
		return nil
	}
	var diags []Diagnostic
	for _, f := range p.Src.Files {
		for _, c := range implementsClauses(f.Src) {
			tObj := p.Lookup(c.typeName)
			if tObj == nil {
				continue // T not in package scope; nothing to verify
			}
			pos := goalPosition(f, c.off)
			for _, iface := range c.ifaces {
				if p.Tables.Sealed[iface] {
					continue // sealed marker method — satisfied by construction
				}
				if d := verifyImplements(p, tObj.Type(), c.typeName, iface, pos); d != nil {
					diags = append(diags, *d)
				}
			}
		}
	}
	return diags
}

// verifyImplements checks that T satisfies the interface named iface. It tests the
// pointer type's method set (the superset that also includes pointer-receiver methods,
// matching goal's `var _ I = (*T)(nil)` assertion form). A nil result means satisfied.
func verifyImplements(p *Package, T types.Type, typeName, iface string, pos token.Position) *Diagnostic {
	it := resolveInterface(p, iface)
	if it == nil {
		return nil // interface unresolvable even via types (e.g. unimported) — leave it
	}
	method, wrongType := types.MissingMethod(types.NewPointer(T), it, true)
	if method == nil {
		return nil // satisfied
	}
	code, what := "unimplemented-method", "missing method"
	if wrongType {
		code, what = "method-signature-mismatch", "wrong signature for method"
	}
	return &Diagnostic{
		Pos:      pos,
		Severity: sema.Error,
		Feature:  "07-implements",
		Code:     code,
		Message: fmt.Sprintf("type `%s` does not implement `%s`: %s `%s`",
			typeName, iface, what, method.Name()),
	}
}

// resolveInterface returns the *types.Interface named by iface, resolving both an
// in-package name ("Speaker") and a qualified one ("io.Writer", via the package's
// imports). It returns nil when the name does not resolve to an interface.
func resolveInterface(p *Package, iface string) *types.Interface {
	var obj types.Object
	if dot := strings.LastIndex(iface, "."); dot >= 0 {
		qual, name := iface[:dot], iface[dot+1:]
		for _, imp := range p.Types.Imports() {
			if imp.Name() == qual {
				obj = imp.Scope().Lookup(name)
				break
			}
		}
	} else {
		obj = p.Lookup(iface)
	}
	if obj == nil {
		return nil
	}
	it, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		return nil
	}
	return it
}

// implClause is one `type T struct implements I, J { … }` clause located in source.
type implClause struct {
	typeName string
	ifaces   []string
	off      int // byte offset of the `implements` keyword
}

// implementsClauses finds every inline implements clause in src (mirrors the lexical 07
// check's locator: the clause sits between `struct` and the body `{`).
func implementsClauses(src string) []implClause {
	toks := scan.Lex(src)
	var out []implClause
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].Text != "type" || !textedit.IsIdent(toks[i+1].Text) || toks[i+2].Text != "struct" {
			continue
		}
		open := -1
		for k := i + 3; k < len(toks); k++ {
			if toks[k].Text == "{" {
				open = k
				break
			}
		}
		if open < 0 {
			continue
		}
		imp := -1
		for k := i + 3; k < open; k++ {
			if toks[k].Text == "implements" {
				imp = k
				break
			}
		}
		if imp < 0 {
			continue
		}
		out = append(out, implClause{
			typeName: toks[i+1].Text,
			ifaces:   splitList(src[toks[imp].End:toks[open].Start]),
			off:      toks[imp].Start,
		})
	}
	return out
}

// splitList splits a comma-separated interface list into trimmed, non-empty names.
func splitList(s string) []string {
	var out []string
	for part := range strings.SplitSeq(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
