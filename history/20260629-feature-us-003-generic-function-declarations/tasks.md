# Implementation Tasks

## Task 1: Add TypeParams to FuncType and walk it
**Status**: completed
**Files**: internal/ast/ast.go, internal/ast/walk.go
**Depends on**: none
**Spec coverage**: FR-1, FR-4 (foundation)
**Verify**: `go build ./...`

### Instructions
- Add `TypeParams *FieldList` to `FuncType` (between `Func` and `Params`),
  documented "generic type parameters in [...]; nil when non-generic".
- In walk.go `*FuncType` case, `if n.TypeParams != nil { Walk(v, n.TypeParams) }`
  before walking Params.

## Task 2: Parse generic func type-param lists
**Status**: completed
**Files**: internal/parser/parser.go
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-2
**Verify**: `go test ./internal/parser/...`

### Instructions
- In `parseFuncDecl`, after `fd.Name = p.ident()` and before
  `ft.Params = p.parseParamList()`, add:
  `if fd.Recv == nil && p.atTypeParams() { ft.TypeParams = p.parseTypeParams() }`.
- Reuses existing helpers used by `parseTypeSpec`.

## Task 3: Emit the func type-param list
**Status**: completed
**Files**: internal/backend/emit.go
**Depends on**: Task 1
**Spec coverage**: FR-3
**Verify**: `go test ./internal/backend/...`

### Instructions
- At the start of `funcSig`, before emitting Params:
  `if t.TypeParams != nil { e.fieldList(t.TypeParams, "[", "]") }`, mirroring
  the TypeSpec emission at emit.go ~347.

## Task 4: Test generic function transpile + build
**Status**: completed
**Files**: internal/backend/*_test.go (existing transpile test file)
**Depends on**: Task 2, Task 3
**Spec coverage**: FR-1, FR-2, FR-3
**Verify**: `task check`

### Instructions
- Add a test transpiling `func Identity[T any](x T) T { return x }` and a
  constrained `func Keys[K comparable, V any](m map[K]V) []K { ... }`,
  asserting the emitted Go contains `[T any]` / `[K comparable, V any]` and that
  the output compiles (assert text, and reuse the go-build harness if present).
