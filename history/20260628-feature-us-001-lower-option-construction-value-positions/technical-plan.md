# Technical Plan — Lower Option construction in value positions

## Overview

Intercept Option construction at the single value-emission seam, `emitter.expr`
(internal/backend/emit.go), so every value position (var/const value, `:=` RHS,
call args, struct fields, slice/map elements) lowers `Option.Some/None` to its
`*T` encoding. Non-addressable `Some(x)` in a pure-expression position is boxed via
a generic helper func injected once per file/package, mirroring `resultPrelude`.

## Components

### internal/backend/lower.go
- `func optionConstruction(x ast.Expr) (kind string, arg ast.Expr, ok bool)` — pure
  classifier: `"none"` for `Option.None`; `"some-addr"` (arg is `*ast.Ident`) for
  `Option.Some(ident)`; `"some-box"` (any other single arg) for `Option.Some(x)`;
  `ok=false` otherwise. Single source of truth shared by the emit branch and the
  prelude scan, so they cannot drift.
- `const optionPrelude` — `func goalSome[T any](v T) *T { return &v }` plus doc
  comment. The boxed-temporary encoding.
- `func needsOptionPrelude(f *ast.File) bool` — `ast.Walk`/`identFinder` scan; true
  iff any node classifies as `"some-box"`. Mirrors `needsFmtImport`.

### internal/backend/emit.go
- `func (e *emitter) tryOptionValue(x ast.Expr) bool` — uses `optionConstruction`;
  emits `nil` / `&<ident>` / `goalSome(<arg>)` (arg via `e.expr`), returns true when
  handled.
- `expr()` gains `if e.tryOptionValue(x) { return }` at the top, before the switch.
- `file()` injects `optionPrelude` (after imports, with the same placement as
  `resultPrelude`) when `!suppressPrelude && needsOptionPrelude(f)`.

### internal/backend/package.go
- `TranspilePackage` appends a shared `goal_options.go` (package clause +
  `optionPrelude`) when any file `needsOptionPrelude`, since per-file emit suppresses
  the inline prelude.

## Interface Contracts

```go
func optionConstruction(x ast.Expr) (kind string, arg ast.Expr, ok bool)
func needsOptionPrelude(f *ast.File) bool
const optionPrelude = "// goalSome boxes v into the *T encoding of Option.Some(v).\nfunc goalSome[T any](v T) *T { return &v }"
func (e *emitter) tryOptionValue(x ast.Expr) bool
```

## Integration Points

- `emitter.expr` (emit.go ~line 820): add the interception line at the top.
- `emitter.file` (emit.go ~line 141): add the inline `optionPrelude` injection
  alongside the existing fmt-import / `resultPrelude` placement.
- `TranspilePackage` (package.go ~line 83): add the shared-file append next to the
  existing `needsResultPrelude` block.
- The existing `optionValueExpr` / `emitOptionReturn` / `emitResultReturn` are left
  unchanged (FR-4); they don't route the whole Option construction through `expr`.

## Testing Strategy

internal/backend/backend_test.go (external `package backend_test`, stdlib `testing`,
NO testify): a new `TestASTEngineLowersOptionInValuePositions` transpiles a file
constructing `Option.Some(v)`/`Option.None` in var-assignment, call-argument,
struct-field, slice-literal, and map-literal positions; asserts `format.Source`
succeeds, no `Option.` token remains, and the `nil` / `&v` encodings appear. Include
a non-addressable `Some(literal)` case to exercise the `goalSome` box. The existing
`TestASTEngineLowersNestedOptionInResult` stays green (FR-4).
