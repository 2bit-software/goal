# Tasks — US-006 Eval variables and assignment

## T1: Add chain-walking Assign to Env
- File: `internal/interp/env.go`
- Add `func (e *Env) Assign(name string, v Value) error`: walk scope chain;
  write to the first scope that already binds `name`; else return
  `&NotFoundError{Name: name}`.
- Verify: `go build ./internal/interp`.
- Depends on: none.

## T2: Variable reads + zero/compound helpers in eval
- File: `internal/interp/eval.go`
- `evalExpr` Ident case: non-true/false → `scope.Lookup(e.Name)`.
- Add `zeroValue(ast.Expr) Value` and `compoundBinOp(token.Kind) (token.Kind,
  bool)`.
- Verify: `go build ./internal/interp`.
- Depends on: none.

## T3: Statement dispatch for declarations and assignment
- File: `internal/interp/interp.go`
- Extend `execStmt` with `*ast.DeclStmt` (VAR/CONST ValueSpecs → Define, zero
  when no initializer) and `*ast.AssignStmt` (DEFINE→Define, ASSIGN→Assign,
  compound→Lookup+applyBinary+Assign; RHS evaluated first; non-ident LHS and
  unsupported ops → descriptive error).
- Verify: `go build ./...` and `go vet ./...`.
- Depends on: T1, T2.

## T4: Tests
- File: `internal/interp/assign_test.go` (package interp)
- AC test (declare via var/`:=`/const, reassign with `=`, compound-assign),
  Assign-updates-existing-binding, undeclared-assign error, undefined-read
  error.
- Verify: `go test ./internal/interp -count=1`, then full
  `go test ./... -count=1`.
- Depends on: T3.
