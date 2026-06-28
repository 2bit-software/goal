# Implementation Plan — US-006 Eval variables and assignment

## Components

### 1. `internal/interp/env.go` — chain-walking assignment
Add `func (e *Env) Assign(name string, v Value) error`: walk the scope chain
(this scope → parents); the first scope whose `vars` contains `name` gets the
new value written in place. If no scope holds the name, return
`&NotFoundError{Name: name}`. This is the deferred US-003 write-to-existing-
binding semantic; `Define` is unchanged (always binds in the current scope).

### 2. `internal/interp/eval.go` — variable reads + zero values
- `evalExpr` `*ast.Ident` case: keep `true`/`false`, and for any other name
  return `scope.Lookup(e.Name)` (propagating `*NotFoundError`).
- Add `zeroValue(typeExpr ast.Expr) Value`: map a declared type ident to a safe
  zero — `int`/integer kinds → `IntVal(0)`, `float*` → `FloatVal(0)`,
  `string` → `StrVal("")`, `bool` → `BoolVal(false)`; otherwise `NilVal()`.
- Add `compoundBinOp(tok token.Kind) (token.Kind, bool)` mapping ADD_ASSIGN→ADD,
  SUB_ASSIGN→SUB, MUL_ASSIGN→MUL, QUO_ASSIGN→QUO, REM_ASSIGN→REM (ok=false for
  anything else → descriptive error at the call site).

### 3. `internal/interp/interp.go` — statement dispatch
Extend `execStmt`:
- `*ast.DeclStmt`: unwrap `*ast.GenDecl`; for `token.VAR`/`token.CONST` iterate
  `*ast.ValueSpec`. For each spec: if `Values` present, evaluate each and
  `Define(name_i, val_i)` pairwise; if absent (var only), `Define(name_i,
  zeroValue(spec.Type))`. (import/type GenDecl as a statement → descriptive
  error; not expected in a function body.)
- `*ast.AssignStmt`: evaluate all `Rhs` first (parallel-assignment order). Then
  for each `Lhs[i]` (an `*ast.Ident`):
  - `token.DEFINE` → `Define(name, rhs_i)`.
  - `token.ASSIGN` → `Assign(name, rhs_i)` (error if undeclared).
  - compound (`compoundBinOp` ok) → `cur, err := scope.Lookup(name)`;
    `applyBinary(op, cur, rhs_i)`; `Assign(name, result)`.
  - else → descriptive "unsupported assignment operator" error.
  A non-ident LHS (`s[i]`, `p.f`) → descriptive error (US-009/US-010 scope).

## Dependency order
env.Assign (1) → eval helpers (2) → execStmt dispatch (3, uses both). No forward
references; each compiles independently.

## Testing
`internal/interp/interp_test.go` (or a new `assign_test.go`, package interp):
- AC test: a program using `var a = 1`, `var b int` (zero), `c := 2`,
  `const d = 10`, then `a = a + b`, `b += 5`, `c -= 1`, `c *= 3`, asserting each
  final value via the run scope.
- `Assign` updates existing binding (not shadow): define in parent, Assign from
  child, read back through parent shows the new value.
- Undeclared assign → `*NotFoundError`. Undefined read → `*NotFoundError`.

## Traceability
- FR-1/FR-3 → DeclStmt VAR/CONST. FR-2 → AssignStmt DEFINE. FR-4 → AssignStmt
  ASSIGN + Env.Assign + RHS-first ordering. FR-5 → compound dispatch +
  applyBinary. FR-6 → Ident Lookup + NotFoundError on assign/read.
