# Implementation Tasks — US-032

## Task 1: Emit expression switch / case clauses
**Status**: completed
**Files**: `internal/backend/emit.go`
**Depends on**: (none)
**Spec coverage**: FR-1 (full ordinary-Go subset incl. switch), FR-4 (goal nodes still gated — unchanged default arms)
**Verify**: `go build ./...`

### Instructions
- In `emit.go`, add to the `stmt(s ast.Stmt)` type switch:
  `case *ast.SwitchStmt: e.switchStmt(s)`.
- Add `switchStmt(s *ast.SwitchStmt)`: emit `"switch "`; if `s.Init != nil`,
  `e.stmt(s.Init)` then `"; "`; if `s.Tag != nil`, `e.expr(s.Tag)` then `" "`;
  then emit the body `"{\n"`, iterate `s.Body.List` calling `caseClause` on each
  `*ast.CaseClause` (fail on any non-CaseClause element), then `"}"`.
- Add `caseClause(c *ast.CaseClause)`: if `len(c.List) > 0` emit
  `"case "` + `exprList(c.List)` + `":\n"`, else `"default:\n"`; then iterate
  `c.Body` calling `e.stmt` + `"\n"` per statement.
- Mirror the structure of the existing `ifStmt`/`forStmt`/`block` helpers. Keep
  the format-once discipline: token-correct Go only; gofmt normalizes layout.
- Refresh the emit.go package/struct doc comment to note US-032 completes the
  ordinary-Go statement set (switch added).

## Task 2: Add full-subset behavioral fixture
**Status**: completed
**Files**: `internal/backend/testdata/plain_full.goal`
**Depends on**: (none)
**Spec coverage**: FR-3 (behavioral conformance witness)
**Verify**: file parses via the emitter in Task 3's test
### Instructions
- Create a plain-Go goal source (no goal-specific constructs) exercising the full
  ordinary-Go subset: a func with an expression `switch` (case + default), a
  struct type decl, a composite struct literal, a map literal, a slice + range,
  a `defer`, a multi-return func, and a const/var declaration.
- Keep it self-contained (stdlib-only, e.g. `fmt`/`strings` or no imports) so the
  corpus temp-module build resolves offline.

## Task 3: Behavioral + switch tests
**Status**: completed
**Files**: `internal/backend/backend_test.go`
**Depends on**: Task 1, Task 2
**Spec coverage**: FR-1, FR-3 (AC: switch transpiles to valid Go; full subset builds+vets)
**Verify**: `go test ./internal/backend/... -count=1`
### Instructions
- Add `TestASTEngineEmitsSwitch`: inline goal source containing a tag+default
  `switch` in a func; call `backend.Transpile`, assert no error, assert
  `go/format.Source` of `out.Go` succeeds, and assert the output contains
  `"switch"`, `"case"`, and `"default"`.
- Add `TestASTEngineBehavioralTierFull`: `-short`-skip; build
  `corpus.Case{ID:"plain-full", Kind:corpus.KindTranspile, Mode:corpus.ModeFile,
  Input:"internal/backend/testdata/plain_full.goal"}` and assert
  `corpus.RunCompile(repoRoot, c, corpus.TranspilerFunc(backend.Transpile))`
  returns nil. Reuse the existing `repoRoot` const and import shape from the
  current backend_test.go.

## Coverage check
- emit.go (plan) → Task 1; plain_full.goal → Task 2; backend_test.go → Task 3.
- FR-1 → Task 1/3; FR-2 → unchanged engine (no task needed); FR-3 → Task 2/3;
  FR-4 → Task 1 (unchanged default arms).
- Full verify gates (`go build ./...`, `go vet ./...`, `go test ./... -count=1`)
  run at the verify step.
