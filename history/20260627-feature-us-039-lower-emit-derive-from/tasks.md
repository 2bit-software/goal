# Implementation Tasks

## Task 1: Port conversion helpers into lower.go
**Status**: completed
**Files**: internal/backend/lower.go
**Depends on**: (none)
**Spec coverage**: FR-4, FR-6
**Verify**: `go build ./internal/backend/...`

### Instructions
Add pure string helpers over resolved type strings, ported from
internal/pass/derive.go but reading sema types:
- `derefType(s string) string`
- `ptrInner(s string) (string, bool)` (handles `*A` and `Option[A]`)
- `arrElem(s string) (n, elem string, ok bool)`
- `mapKV(s string) (k, v string, ok bool)`
- `elemConv(a, b string, reg map[[2]string]sema.ConvEntry) (func(string) string, error)`
- `findSemaField(fields []sema.Field, name string) (sema.Field, bool)` (case-insensitive)
- `deriveTarget(... )` equivalent: split a result list into (tgtType string, fallible bool).

## Task 2: Add deriveDecl + genConversion + resolveField to emit.go
**Status**: completed
**Files**: internal/backend/emit.go
**Depends on**: Task 1
**Spec coverage**: FR-2, FR-3, FR-4, FR-5, FR-6, error handling
**Verify**: `go build ./internal/backend/...`

### Instructions
- In `funcDecl`, before the gensym-scope setup, add `case ast.FuncDerive` to the
  `d.Mod` switch: call `e.deriveDecl(d)` and `return`.
- `deriveDecl(d *ast.FuncDecl)`: extract source name + type (first param), target
  type + fallibility (result list via the Task 1 splitter), and overrides from the
  body. Emit `func name(src S) <ret> { ... }` via `genConversion`.
- `deriveOverrides(body)`: walk `Body -> ReturnStmt -> CompositeLit.Elts`; a
  `KeyValueExpr{Key: Ident, Value}` is an override (Value `Ident "_"` => skip); a
  `SpreadElement{X: CallExpr{Fun: Ident "derive"}}` is the fill marker.
- `genConversion`: `var out T`; emit overrides first (skip `_`), then for each
  unoverridden target field resolve via `resolveField`; `return out` (or
  `return out, nil` when fallible).
- `resolveField(dst, srcExpr, sf, tf, fallibleOK, errName)`: identity / registry
  total / registry fallible / pointer-or-Option / slice / array / map / nested
  struct, mirroring internal/pass/derive.go. Temp names via `e.gensym`. An
  unresolvable field => `e.fail(...)` with a located message (and return).
- Override values emit via `e.exprText(value)` (AST expr, not string parse).

## Task 3: Tests for derive behavioral tier + encoding
**Status**: completed
**Files**: internal/backend/backend_test.go
**Depends on**: Task 2
**Spec coverage**: all acceptance criteria
**Verify**: `go test ./internal/backend/ -run TestASTEngineDerive -count=1`

### Instructions
- Add `deriveCases` = the 3 features/12 inputs.
- `TestASTEngineDeriveBehavioralTier`: `-short`-skip; each case through
  `backend.Transpile` + `corpus.RunCompile` (build+vet).
- `TestASTEngineDeriveEncoding`: assert `func uuidToString(` (from-strip), identity
  assignment, total-leaf call, fallible early-return shape, and slice make+loop.

## Task 4: Full verify, flip prd, log progress
**Status**: completed
**Files**: prd.json, progress.txt
**Depends on**: Task 3
**Spec coverage**: gate
**Verify**: `go build ./...` && `go vet ./...` && `go test ./... -count=1`
