package typecheck

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"goal/internal/check"
)

// CheckNoZeroValue is the depth version of feature 08 (no-zero-value) for the residual
// the lexical check (internal/check/fields.go) cannot reach: an *elided* composite
// literal — one that omits its type because Go infers it from the surrounding array,
// slice, or map type (`[]Inner{{a: 1}}`, `map[string]Inner{"k": {a: 1}}`,
// `[N]Inner{{a: 1}}`). Such a literal is valid Go that silently zero-fills any omitted
// field — exactly the footgun feature 08 exists to close — yet its required-field set is
// invisible to a lexical scan:
//
//   - The lexical check only fires on `T{…}` where the type token `T` precedes the brace
//     at the site (its documented defer-boundary). An element literal has no type token,
//     so the surrounding `Inner{` of `[]Inner{{…}}` is what the scan latches onto — and,
//     unable to see into the nested element, it reports the WRONG field set (every field
//     "missing", even those the element actually supplies). go/types reads the element's
//     inferred type and reports the field-accurate set instead.
//
// This check therefore runs only on AST composite literals with no type expression
// (lit.Type == nil) that go/types resolves to a struct declared in THIS goal package (so
// the goal guarantee genuinely applies — an imported Go struct carries no such promise).
// It is the type-backed answer for the same construct; when both stages flag one literal,
// the CLI merge prefers this one (DEPTH-TODO open decision: dedup → type-backed).
func CheckNoZeroValue(p *Package) []Diagnostic {
	if p.Types == nil {
		return nil
	}
	var diags []Diagnostic
	for _, f := range p.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if cl.Type != nil {
				return true // type named at the site — the lexical stage owns it
			}
			if d := elidedLitDiag(p, cl); d != nil {
				diags = append(diags, *d)
			}
			return true
		})
	}
	return diags
}

// elidedLitDiag returns a missing-field Error for an elided composite literal cl that
// resolves to a goal-package struct and omits at least one required field. It returns nil
// when cl is not such a literal (unresolved type, non-struct, out-of-package, positional,
// or complete) — a false "complete" is worse than an honest skip, and an unresolved
// elided literal is already a Go error the harness collected.
func elidedLitDiag(p *Package, cl *ast.CompositeLit) *Diagnostic {
	st, name := goalStructLit(p, cl)
	if st == nil {
		return nil
	}
	present, keyed := litFieldKeys(cl)
	if !keyed {
		return nil // positional element — Go itself requires every field, so no goal gap
	}
	missing := missingStructFields(st, present)
	if len(missing) == 0 {
		return nil
	}
	return &Diagnostic{
		Pos:      p.Fset.Position(cl.Pos()),
		Severity: check.Error,
		Feature:  "08-no-zero-value",
		Code:     "elided-missing-field",
		Message: fmt.Sprintf(
			"elided literal of `%s` omits required field%s %s — its type is inferred from the surrounding collection, so name every field (or write `%s{…, ...defaults}`)",
			name, plural(len(missing)), quoteJoin(missing), name),
	}
}

// goalStructLit resolves cl's inferred type and returns the underlying struct and its
// goal name when it is a named struct declared in this goal package; otherwise nil. The
// package-identity guard keeps the check from asserting feature 08's guarantee on an
// imported Go struct (which carries no such promise) or an injected helper type.
func goalStructLit(p *Package, cl *ast.CompositeLit) (*types.Struct, string) {
	tv, ok := p.Info.Types[cl]
	if !ok || tv.Type == nil {
		return nil, "" // unresolved (already a Go error) — defer, don't guess
	}
	named, ok := tv.Type.(*types.Named)
	if !ok || named.Obj().Pkg() != p.Types {
		return nil, ""
	}
	name := named.Obj().Name()
	if _, isGoalStruct := p.Tables.Structs[name]; !isGoalStruct {
		return nil, "" // not a goal-declared struct (e.g. an injected sum type)
	}
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, ""
	}
	return st, name
}

// litFieldKeys returns the set of field names a composite literal names by key, and
// whether the literal is keyed at all. A literal with any non-keyed element is positional
// (keyed=false): Go already requires every field of a positional struct literal, so it is
// not a goal-specific gap. An empty literal `{}` is keyed-vacuous (keyed=true, no keys),
// which correctly reports every field missing.
func litFieldKeys(cl *ast.CompositeLit) (present map[string]bool, keyed bool) {
	present = map[string]bool{}
	for _, e := range cl.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		if !ok {
			return nil, false
		}
		id, ok := kv.Key.(*ast.Ident)
		if !ok {
			return nil, false
		}
		present[id.Name] = true
	}
	return present, true
}

// missingStructFields returns the struct's field names not present in the literal, in
// declaration order. Embedded fields are required like any other named field.
func missingStructFields(st *types.Struct, present map[string]bool) []string {
	var missing []string
	for i := 0; i < st.NumFields(); i++ {
		if name := st.Field(i).Name(); !present[name] {
			missing = append(missing, name)
		}
	}
	return missing
}

// plural returns "s" when n != 1, for "field"/"fields" agreement.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// quoteJoin renders names as a comma-separated list of backtick-quoted identifiers in the
// order given.
func quoteJoin(names []string) string {
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = "`" + n + "`"
	}
	return strings.Join(quoted, ", ")
}
