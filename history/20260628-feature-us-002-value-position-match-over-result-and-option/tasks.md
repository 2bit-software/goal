# Implementation Tasks — US-002

## Task 1: Generalize Result/Option match lowering to accept a match position
**Status**: completed
**Files**: internal/backend/emit.go
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2, FR-3
**Verify**: `go build ./...`

### Instructions
- Change `resultMatch`, `closedResultMatch`, `optionMatch` signatures to
  `(m *ast.MatchExpr, pos matchPos, name string)`.
- Replace direct `armBody(...)` calls with `armWrap(body, pos, name)`; replace
  `armBodyRenamed(body, binding, target)` with a new
  `armBodyRenamedWrap(body, binding, target, pos, name)` that scopes the rename and
  then calls `armWrap`.
- Update `matchStmt` to call them with `(m, posStmt, "")` (statement position
  unchanged).

## Task 2: Wire value-position dispatch
**Status**: completed
**Files**: internal/backend/emit.go
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-2
**Verify**: `go build ./...`

### Instructions
- `returnStmt`: extend the single-result MatchExpr branch so a `Result` qualifier
  routes to `resultMatch(m, posReturn, "")` and `Option` to
  `optionMatch(m, posReturn, "")` (keep the enum branch).
- `tryVarMatch`: accept a Result/Option match value (check qualifier before
  emitting `var name T`), dispatch with posVar.
- `tryAssignMatch`: accept a Result/Option match RHS; reuse `inferMatchType` for
  the `var name T` type, dispatch with posVar; non-inferable keeps the located
  deferral.

## Task 3: Backend test
**Status**: completed
**Files**: internal/backend/backend_test.go
**Depends on**: Task 2
**Spec coverage**: AC backend test, FR-3, FR-4
**Verify**: `go test ./internal/backend/ -count=1`

### Instructions
- Add a test transpiling value-position match in: Result/return, Option/assign,
  Option/return. Use call subjects (`parse(input)` Result, `find(id)` Option) like
  features/03-result and features/04-option.
- Assert no transpile error, valid Go (build under temp module or go/format), and
  both arm bodies present in output.

## Task 4: Full verify
**Status**: completed
**Depends on**: Task 3
**Verify**: `go build ./...`, `go vet ./...`, `go test ./... -count=1`
