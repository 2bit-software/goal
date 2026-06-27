# Research — US-006 Eval variables and assignment

This is internal interpreter work over the project's own shared AST + Env; no
third-party libraries or external approaches are relevant. The design follows
established in-repo patterns rather than outside references.

## Findings (from the codebase)

- Statement dispatch already exists as `Interp.execStmt` (interp.go). The two
  new statement kinds are `*ast.DeclStmt` (wraps a `*ast.GenDecl` with
  `Tok == token.VAR | token.CONST` and `*ast.ValueSpec` specs) and
  `*ast.AssignStmt` (`Tok` ∈ {DEFINE, ASSIGN, ADD_ASSIGN, ...}).
- The Env scope chain (env.go) supplies `Define` (bind in current scope) and
  `Lookup` (walk chain). It is MISSING a chain-walking write — US-003's note
  explicitly deferred "write to an existing outer binding vs. define-in-place"
  to US-006. So add `Env.Assign(name, v) error` that finds the owning scope and
  overwrites it, returning `*NotFoundError` when undeclared.
- `evalExpr`'s `*ast.Ident` case (eval.go) errors for any non true/false ident;
  it must instead `scope.Lookup(e.Name)`.
- `applyBinary` (eval.go) already implements `+ - * / %` etc., so compound
  assignment reuses it via a token→binary-op map (ADD_ASSIGN→ADD, ...). For the
  bitwise/shift compound forms not yet in `applyBinary`, return a descriptive
  named error rather than a silent result (Go-subset corpus does not exercise
  them yet; arithmetic compounds are the AC focus).
- `var x T` with no initializer needs a zero Value. v1 zero: int→0, float→0.0,
  string→"", bool→false; otherwise KindNil. Keep it minimal — composite zero
  values arrive with US-009.

## Decisions

- Plain `=` and compound assigns use `Env.Assign` (mutate existing binding).
- `:=`, `var`, `const` use `Env.Define` (bind in current scope).
- Multi-name / multi-value specs and parallel assignment (`a, b = b, a`)
  evaluate all RHS before binding, matching Go.

## Confidence

High — entirely in-repo, mirrors the US-005 eval seam and US-003 Env design.
