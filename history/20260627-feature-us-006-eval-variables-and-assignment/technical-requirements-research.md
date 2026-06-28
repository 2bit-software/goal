# Technical Requirements / Research — US-006

## Existing seams

- `internal/interp/interp.go` — `execStmt` is the statement-dispatch seam
  (currently handles `*ast.ExprStmt` and `*ast.EmptyStmt`). Add `*ast.DeclStmt`
  (const/var) and `*ast.AssignStmt` (`:=`, `=`, compound) cases here.
- `internal/interp/eval.go` — `evalExpr`'s `*ast.Ident` case currently only
  handles predeclared `true`/`false` and errors otherwise; route other idents
  through `scope.Lookup`. `applyBinary` already implements the arithmetic the
  compound assigns reuse.
- `internal/interp/env.go` — add an `Assign(name, v)` method that walks the
  scope chain and overwrites the binding in the scope where it is defined
  (vs `Define`, which always binds in the current scope). Plain/compound `=`
  use `Assign`; `:=` and `var`/`const` use `Define`.

## AST facts

- `var`/`const` are `*ast.DeclStmt{Decl: *ast.GenDecl{Tok: VAR|CONST,
  Specs: []*ast.ValueSpec}}`. A `ValueSpec` has `Names []*Ident`,
  `Type Expr` (optional), `Values []Expr` (optional for var → zero value).
- `:=` / `=` / compound are `*ast.AssignStmt{Lhs, Tok, Rhs}`. `Tok` is
  `token.DEFINE`, `token.ASSIGN`, or `token.ADD_ASSIGN`/`SUB_ASSIGN`/... .
- Compound-assign token → binary op mapping: ADD_ASSIGN→ADD, SUB_ASSIGN→SUB,
  MUL_ASSIGN→MUL, QUO_ASSIGN→QUO, REM_ASSIGN→REM (plus bitwise/shift forms).

## Test approach

`package interp` unit test parses + sema-resolves a small program, runs it,
and asserts the final variable values (read back via the interpreter's root/
child scope). A bare expression statement in a body reaches `evalExpr` via
`findMain().Body`.
