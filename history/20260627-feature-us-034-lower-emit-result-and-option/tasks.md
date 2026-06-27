# Implementation Tasks — US-034

## Task 1: Add Result/Option encoders and helpers to lower.go
**Status**: completed
**Files**: `internal/backend/lower.go`
**Depends on**: (none)
**Spec coverage**: FR-1..FR-6 (shared helpers)
**Verify**: `go build ./internal/backend/...`

### Instructions
- Add the gensym name constants: `okName="__goal_ok"`, `errName="__goal_err"`,
  `valName="__goal_v"`, `someName="__goal_some"`, `optBase="__goal_o"` (mirror
  `internal/pass/pass.go`).
- Add `roKind` (roNone/roResultOpen/roOption) and
  `resultOptionKind(t *ast.FuncType) (roKind, ast.Expr)`: a single unnamed result
  whose type is `*ast.IndexListExpr{X: Ident "Result", Indices:[T,E]}` with
  `isErrorIdent(E)` -> (roResultOpen, T); `*ast.IndexExpr{X: Ident "Option"}` ->
  (roOption, Index); else (roNone, nil).
- Add `isErrorIdent(ast.Expr) bool`, `matchQualifier(*ast.MatchExpr) string`
  (first arm `*ast.VariantPattern`'s Enum `*ast.Ident` name; "" otherwise), and
  `usesIdent(ast.Node, string) bool` (ast.Walk for an `*ast.Ident` of that name).
- Keep the encoders text-only / format-once discipline.

## Task 2: Wire signature, return-constructor, Option-type, and match lowering into emit.go
**Status**: completed
**Files**: `internal/backend/emit.go`
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-2, FR-3, FR-4, FR-5, FR-6
**Verify**: `go build ./... && go vet ./...`

### Instructions
- Add `fnKind roKind` and `renames map[string]string` fields to `emitter`.
- `funcDecl`: compute `resultOptionKind(d.Type)`; save/restore `e.fnKind` around
  the body emission.
- `funcSig`: when the single unnamed result classifies as roResultOpen, emit
  `(__goal_ok <T>, __goal_err error)` (T via `e.expr`); roOption falls through to
  the `IndexExpr` lowering below (so the result renders `*T`).
- `expr` `*ast.IndexExpr`: when `X` is `*ast.Ident{Name:"Option"}`, emit `*` then
  `e.expr(Index)`.
- `expr` `*ast.Ident`: if `e.renames[name]` is set, emit the renamed value.
- `stmt` `*ast.ReturnStmt`: when `len(Results)==1` and the single result is a
  `Result.Ok/Err(...)` call and `e.fnKind==roResultOpen` -> emit `return X, nil`
  / `return __goal_ok, X`. When it is `Option.None` (SelectorExpr) /
  `Option.Some(x)` (CallExpr) and `e.fnKind==roOption` -> emit `return nil` /
  `return &x` (identifier) or box (`__goal_some := x` newline `return &__goal_some`).
- `stmt` `*ast.ExprStmt`: when `X` is `*ast.MatchExpr`, dispatch on
  `matchQualifier`: "Result" -> result match lowering (guard: if the scrutinee
  callee's `sema.Info.FuncSignatures` mode is `ModeResultClosed`, `e.fail` —
  closed-E is US-037); "Option" -> option match lowering; else fall through to
  `e.expr(s.X)` (value-position match is US-036 and will error there as before).
- Result match lowering: find Ok/Err arms by variant; `okUsed :=
  usesIdent(okArm.Body, okBinding)`; emit
  `<lhs>, __goal_err := <scrut>` then `if __goal_err != nil { <errBody> } else {
  <okBody> }`, setting `renames[okBinding]=valName` for the Ok body and
  `renames[errBinding]=errName` for the Err body (clear after each).
- Option match lowering: emit `if __goal_o := <scrut>; __goal_o != nil {` then
  (if Some binding present and used) `<bind> := *__goal_o` then `<someBody>`,
  `} else {` `<noneBody>` `}` — no rename.
- Arm body emission helper: a `*ast.BlockStmt`/`ast.Stmt` -> `e.stmt`, an
  `ast.Expr` -> `e.expr` (expression statement).

## Task 3: Add the Result/Option behavioral-tier test
**Status**: completed
**Files**: `internal/backend/backend_test.go`
**Depends on**: Task 2
**Spec coverage**: AC (behavioral tier over 03-result/04-option)
**Verify**: `go test ./internal/backend/ -run ResultOption -count=1`

### Instructions
- Add `TestASTEngineResultOptionBehavioralTier` (external `backend_test` pkg):
  enumerate the 03-result and 04-option example inputs (a literal list mirroring
  `enumsImplementsCases`, or filter the manifest), run each through
  `corpus.RunCompile(repoRoot, c, corpus.TranspilerFunc(backend.Transpile))` in a
  subtest; `-short`-skipped; `t.Fatal` on zero cases.
- Optionally add a focused `TestASTEngineResultOptionEncoding` asserting a Result
  function emits a `, error)` return + `, nil` ok path, and an Option function
  emits `*int` + `return nil`.

## Task 4: Run full verify gates, flip prd, log progress
**Status**: completed
**Files**: `prd.json`, `progress.txt`
**Depends on**: Task 3
**Spec coverage**: all (gate)
**Verify**: `go build ./... && go vet ./... && go test ./... -count=1`
