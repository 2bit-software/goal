# Technical Plan — SEAM-CAP-2

## Overview

Two-part change, mirrored in internal/ (live) and selfhost/ (.goal):
1. Foreign enrichment reads sibling `.goal` source for exported enums.
2. Backend lowers package-qualified bare variant construction to §8.1 form.

## Components & Changes

### C1 — Foreign enrichment (`internal/sema/foreign.go`)
- Add `import "goal/internal/parser"`.
- In `foreignDecls`, classify dir entries into non-test `.go` and `.goal`. When there are
  no `.go` files but there are `.goal` files, delegate to a new `goalForeignDecls`.
- New `goalForeignDecls(dir, requestedAlias string, goalFiles []string)` with the same
  5-return signature: parse each `.goal` via `parser.ParseFile`, derive the alias from the
  package clause when `requestedAlias` is empty, `ResolvePackage` the files, then project
  EXPORTED enums into `enums` keyed `alias.Enum` with requalified variant field types.
  Structs/funcs/methods are returned empty (enum-only scope).
- New helper `isExportedName(name string) bool` (first rune upper) and
  `qualifyForeignType(t, alias string) string` (best-effort: peel `*`, `[]`, `[N]`,
  `map[..]`, `...`; prefix a bare non-builtin core identifier with `alias.`).

### C2 — Backend construction lowering (`internal/backend/lower.go`, `emit.go`)
- Add `enumRef(x ast.Expr) (key string, ok bool)` in lower.go: `*ast.Ident` -> name;
  `*ast.SelectorExpr{X:*ast.Ident, Sel}` -> `pkg.Sel`.
- `emit.go` `selectorExpr`: use `enumRef(x.X)` to resolve the enum key; on a resolved enum
  whose `VSet` contains `x.Sel.Name`, emit `key(key_Variant{})`.
- `emit.go` `variantLit`: use `enumRef(x.Enum)` for the key (supports payload cross-pkg).
- `emit.go` `armBodyType`: use `enumRef(b.X)` so a whole-body cross-pkg variant ref infers
  the qualified enum type.

### C3 — selfhost mirror
- Apply C1 to `selfhost/sema/foreign.goal` (import `goal/selfhost/parser`).
- Apply C2 to `selfhost/backend/lower.goal` and `selfhost/backend/emit.goal`.

## Interface Contracts

```go
// internal/sema/foreign.go
func goalForeignDecls(dir, requestedAlias string, goalFiles []string) (
    structs map[string][]Field, funcs, methods map[string]FuncSig,
    enums map[string]*Enum, err error)
func isExportedName(name string) bool
func qualifyForeignType(t, alias string) string

// internal/backend/lower.go
func enumRef(x ast.Expr) (string, bool)
```

## Integration Points

- `foreignDecls` is called only from `EnrichForeign` (foreign.go) — no caller signature
  change. `goalForeignDecls` returns the same tuple shape, so `EnrichForeign`'s merge loop
  is untouched.
- `enumRef` is consumed inside emit.go enum-construction sites; match lowering already
  handles the qualified case via `matchQualifier` + `enumOf` (SEAM-CAP).

## Testing Strategy

- New fixture: `internal/backend/testdata/goalenum/mood/mood.goal` (defines an enum +
  function) and `.../goalenum/use/use.goal` (cross-package `match` + bare construction),
  both `module goal`-relative so `moduleResolve` resolves the import.
- New test `internal/backend/crosspkg_goal_enum_test.go`:
  - Transpile `use` via `backend.TranspilePackage` (real per-package topology); assert the
    emitted Go contains the §8.1 type-switch (`case mood.Mood_*:`) and the construction
    form `mood.Mood(mood.Mood_*{})`.
  - Behavioral: transpile BOTH packages, write generated Go + a reference-switch test into
    a temp `module goal`, `go test` to prove identical behavior.
- Gates: `task check`, `task build`, `task fixpoint`; confirm corpus behavioral tier
  unchanged (additive fixture only).
