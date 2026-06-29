# Technical Requirements / Research

## Root cause

- `internal/backend/emit.go` `sealedInterfaceDecl` (~L240) calls
  `genSealedInterface(d.Name.Name)` and ignores `d.Methods` (a parsed
  `*ast.FieldList` already present on `ast.SealedInterfaceDecl`).
- `internal/backend/lower.go` `genSealedInterface` (~L297) emits only the marker.
- Identical mirror in `selfhost/backend/emit.goal` and `selfhost/backend/lower.goal`.

## Approach

- When `d.Methods` is empty/nil: keep `genSealedInterface` (compact marker-only form)
  so existing output stays byte-identical (fixpoint safe).
- When `d.Methods` has entries: emit `type Name interface {` + each declared method
  (reuse the existing interface-method rendering used by `interfaceType`) + the
  marker `isName()` + `}`. Output is gofmt-normalized downstream.
- Reuse the method-emit loop from `interfaceType` (extract a small helper to avoid
  duplication).
- Mirror exactly in selfhost/backend/emit.goal (the .goal source is a Go superset).

## Verification

- `task check`, `task build`, `task fixpoint` (watch fixpoint — touches emit/lower).
- New regression test in internal/backend: a sealed interface declaring methods
  keeps them + the marker in emitted Go, and an implementor calling through the
  interface builds (proves methods preserved + callable).
