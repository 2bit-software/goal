package typecheck

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"
	"strings"

	"goal/internal/analyze"
	"goal/internal/check"
)

// CheckMustUse is the depth version of feature 03's must-use guarantee. The lexical
// check (internal/check/mustuse.go) catches only the statement-leading dropped call
// (`parse(input)` on its own line) and explicitly refuses the assigned/stored cases as
// its "go/types graduation boundary" (DECISIONS §03). Once a Result/Option lowers to Go,
// Go's own "declared and not used" check already rejects the simplest dropped local — so
// re-catching that adds nothing. This check targets the two genuinely-deferred flow
// subsets that Go does NOT catch and types CAN resolve:
//
//   - discarded-result-error: `v, _ := f()` / `_, _ := f()` where f is an open-E Result
//     function (lowered to a (T, error) tuple). The error channel is dropped via `_` —
//     legal Go, but the must-use violation goal exists to prevent. (Closed-E Result and
//     Option lower to a single value, so their `v, _ :=` form is a Go assignment mismatch
//     Go already rejects — not in scope here.)
//   - dropped-stored-result: a Result/Option-typed struct field (goal-typed per the
//     tables) that is never consulted via a selector anywhere in the package. Go does not
//     flag unused struct fields. An unexported field is package-private, so a never-read
//     one is provably dropped (Error); an exported one cannot be proven unread elsewhere,
//     so a never-read-in-package exported field is an honest deferral (Warning).
//
// The goal tables locate the constructs (which functions are Result-mode, which fields
// are Option/Result-typed); go/types decides the flow question (is the error position
// blank, is the field var ever read via a selection).
func CheckMustUse(p *Package) []Diagnostic {
	if p.Types == nil {
		return nil
	}
	var diags []Diagnostic
	diags = append(diags, checkDiscardedError(p)...)
	diags = append(diags, checkDroppedField(p)...)
	return diags
}

// checkDiscardedError flags `v, _ := f()` (and `_, _ := f()`) where f is an open-E
// Result function and the error return is discarded with the blank identifier.
func checkDiscardedError(p *Package) []Diagnostic {
	var diags []Diagnostic
	for _, f := range p.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			if as, ok := n.(*ast.AssignStmt); ok {
				if d := discardedResultError(p, as); d != nil {
					diags = append(diags, *d)
				}
			}
			return true
		})
	}
	return diags
}

// discardedResultError returns a diagnostic when as binds an open-E Result call's two
// returns and discards the error (last) position with `_`. A nil result means clean.
func discardedResultError(p *Package, as *ast.AssignStmt) *Diagnostic {
	// Open-E Result lowers to exactly (T, error); the tuple-destructure has one call RHS
	// and two LHS. Any other arity is a different (or Go-rejected) construct.
	if len(as.Lhs) != 2 || len(as.Rhs) != 1 {
		return nil
	}
	call, ok := as.Rhs[0].(*ast.CallExpr)
	if !ok {
		return nil
	}
	// Resolve the callee by goal name; only an in-package identifier maps reliably to a
	// table signature (a selector callee is deferred silently, like the §02 boundary).
	callee, ok := call.Fun.(*ast.Ident)
	if !ok {
		return nil
	}
	sig, ok := p.Tables.FuncSignatures[callee.Name]
	if !ok || sig.Mode != analyze.ModeResult {
		return nil
	}
	if !isBlank(as.Lhs[1]) {
		return nil // error position consulted (or named for later use, which Go enforces)
	}
	return &Diagnostic{
		Pos:      p.Fset.Position(as.Lhs[1].Pos()),
		Severity: check.Error,
		Feature:  "03-result",
		Code:     "discarded-result-error",
		Message: fmt.Sprintf(
			"the error from `%s` is discarded with `_`; consume the Result with `match`, `?`, or bind and inspect the error",
			callee.Name),
	}
}

// checkDroppedField flags Result/Option-typed struct fields that are never read via a
// selector. Unexported never-read fields are an Error (provably dropped, package-private);
// exported never-read-in-package fields are a deferral Warning (may be read elsewhere).
//
// It iterates the real go/types fields of each goal-declared struct (so the analysis is
// grounded in resolved types, not the source text) and decides must-use-ness per field:
// a closed-E Result field is the injected `Result` named type (read straight from
// go/types); an Option field lowers to an ambiguous `*T`, so its goal declaration is
// confirmed against the tables by name.
func checkDroppedField(p *Package) []Diagnostic {
	consulted := consultedFields(p)

	names := make([]string, 0, len(p.Tables.Structs))
	for name := range p.Tables.Structs {
		names = append(names, name)
	}
	sort.Strings(names) // deterministic diagnostic order

	var diags []Diagnostic
	for _, sname := range names {
		st := structType(p, sname)
		if st == nil {
			continue
		}
		for i := 0; i < st.NumFields(); i++ {
			v := st.Field(i)
			kind, ok := mustUseFieldKind(p, sname, v)
			if !ok || consulted[v] {
				continue
			}
			// A field whose lowered type did not resolve (e.g. an open-E Result field,
			// which has no single-value lowering) yields no sound verdict — defer.
			if !isValid(v.Type()) {
				continue
			}
			diags = append(diags, droppedFieldDiag(p, sname, v.Name(), kind, v))
		}
	}
	return diags
}

// mustUseFieldKind reports whether field v of struct sname is a must-use sum type, naming
// it ("Result" or "Option"). A Result field is recognized from its resolved type (the
// injected generic sum); an Option field lowers to a bare pointer, so its goal-declared
// `Option[...]` type is confirmed against the tables by field name.
func mustUseFieldKind(p *Package, sname string, v *types.Var) (kind string, ok bool) {
	if isResultNamed(p, v.Type()) {
		return "Result", true
	}
	if optionDeclared(p, sname, v.Name()) {
		return "Option", true
	}
	return "", false
}

// isResultNamed reports whether t is the package's injected `Result[T, E]` sum type (the
// closed-E Result encoding), possibly instantiated.
func isResultNamed(p *Package, t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Name() == "Result" && obj.Pkg() == p.Types
}

// optionDeclared reports whether struct sname declares field fname as a goal `Option[...]`
// type, per the tables. (The struct-body table parser is reliable for an Option field's
// single type argument; a multi-argument `Result[T, E]` line it mis-splits is recognized
// from go/types instead — see isResultNamed.)
func optionDeclared(p *Package, sname, fname string) bool {
	for _, f := range p.Tables.Structs[sname] {
		if f.Name == fname && strings.HasPrefix(strings.TrimSpace(f.Type), "Option[") {
			return true
		}
	}
	return false
}

// droppedFieldDiag builds the Error (unexported) or deferral Warning (exported) for a
// never-consulted must-use field.
func droppedFieldDiag(p *Package, sname, fname, kind string, v *types.Var) Diagnostic {
	pos := p.Fset.Position(v.Pos())
	if v.Exported() {
		return Diagnostic{
			Pos: pos, Severity: check.Warning, Feature: "03-result",
			Code: "unresolved-dropped-field",
			Message: fmt.Sprintf(
				"exported field `%s.%s` holds a %s but is never read in this package; cannot prove it is consulted elsewhere",
				sname, fname, kind),
		}
	}
	return Diagnostic{
		Pos: pos, Severity: check.Error, Feature: "03-result",
		Code: "dropped-stored-result",
		Message: fmt.Sprintf(
			"field `%s.%s` stores a %s that is never consulted; match it, propagate it, or read it",
			sname, fname, kind),
	}
}

// consultedFields returns the set of struct-field objects read via a selector anywhere in
// the package. A composite-literal key (the store) is not a selection, so storing a value
// into a field does not, by itself, count as consulting it.
func consultedFields(p *Package) map[*types.Var]bool {
	consulted := map[*types.Var]bool{}
	for _, sel := range p.Info.Selections {
		if v, ok := sel.Obj().(*types.Var); ok && v.IsField() {
			consulted[v] = true
		}
	}
	return consulted
}

// structType returns the *types.Struct underlying the package-scope type named name, or
// nil when name is not an in-package struct.
func structType(p *Package, name string) *types.Struct {
	obj := p.Lookup(name)
	if obj == nil {
		return nil
	}
	st, _ := obj.Type().Underlying().(*types.Struct)
	return st
}

// isBlank reports whether e is the blank identifier `_`.
func isBlank(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "_"
}

// isValid reports whether t is a resolved (non-invalid) type.
func isValid(t types.Type) bool {
	b, ok := t.(*types.Basic)
	return !ok || b.Kind() != types.Invalid
}
