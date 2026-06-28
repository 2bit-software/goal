# Tasks — US-007 Eval functions and calls

Ordered so each task only depends on completed prior tasks; the package compiles
after each.

## Task 1 — FuncValue carries its declaration
- **Files**: `internal/interp/value.go`
- **Do**: Extend `FuncValue` with `Decl *ast.FuncDecl` and `Env *Env` fields
  (keep `Name`). Add constructor `FuncDeclVal(decl *ast.FuncDecl, env *Env) Value`
  setting Name from `decl.Name`. Keep the existing `FuncVal(name)` constructor.
  Import `goal/internal/ast`.
- **Covers**: FR-1.
- **Verify**: `go build ./internal/interp` and `go test ./internal/interp -run TestConstructEachKind` still pass.

## Task 2 — Register top-level functions; return signal
- **Files**: `internal/interp/interp.go`
- **Do**: Add `type returnSignal struct { vals []Value }` implementing `error`.
  In `New`, walk `file.Decls` and `root.Define(name, FuncDeclVal(fn, root))` for
  each plain func (Recv==nil, Name!=nil, Body!=nil). `Run` looks up `main` and
  calls it (via the call helper from Task 3) — for now keep running main's body
  through a child scope, intercepting a returnSignal as success.
- **Covers**: FR-1, FR-5, FR-7.
- **Verify**: `go build ./internal/interp`; existing interp_test still green.

## Task 3 — Call evaluation + control flow
- **Files**: `internal/interp/eval.go`, `internal/interp/interp.go`
- **Do**: Add `evalCallMulti(call, scope) ([]Value, error)`: resolve callee
  (Ident->Lookup) to KindFunc (else "cannot call <kind>"), evaluate args, flatten
  declared params from `Type.Params.List[].Names`, arity-check, bind in a fresh
  child of the func's `Env`, run body via `execBlock`, recover `returnSignal` for
  the result (`errors.As`). Add `CallExpr` case to `evalExpr` (require exactly 1
  result). Add `execStmt` cases: `*ast.ReturnStmt` (eval results -> returnSignal),
  `*ast.IfStmt` (bool cond, child-scoped Body/Else, Init first). Add the
  multi-result branch to `execAssign` (`len(Lhs)>len(Rhs)==1` + sole RHS CallExpr).
- **Covers**: FR-2, FR-3, FR-4, FR-6.
- **Verify**: `go build ./... && go vet ./...`.

## Task 4 — Tests
- **Files**: `internal/interp/call_test.go` (new)
- **Do**: `package interp` tests — recursive factorial(5)==120, fib(10)==55
  (parse+sema-resolve a goal program, `New`, evaluate a `CallExpr` against root,
  assert Value), multi-return `q,r := divmod(17,5)`, arg-count mismatch error,
  non-function call error, undefined call error. stdlib testing, no testify.
- **Covers**: all ACs.
- **Verify**: `go test ./... -count=1`.

## Coverage check
- FR-1: T1,T2 · FR-2: T3 · FR-3: T3 · FR-4: T3 · FR-5: T2 · FR-6: T3 · FR-7: T2,T4
- Files: value.go(T1), interp.go(T2,T3), eval.go(T3), call_test.go(T4) — all in plan inventory.
