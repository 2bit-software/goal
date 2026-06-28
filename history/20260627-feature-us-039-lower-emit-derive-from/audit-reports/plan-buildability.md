# Plan Audit — Buildability

- Dependency order is valid: pure lower.go helpers first, then emitter methods that
  use them, then the funcDecl dispatch, then tests. No forward references.
- Interface contracts agree: `resolveField` returns `[]string` statements emitted by
  `genConversion`; helpers take/return strings sourced from `sema.Field.Type` and
  `sema.ConvEntry`, which already exist.
- File paths are real (`internal/backend/emit.go`, `lower.go`, `backend_test.go`).
- Integration point is specific: `funcDecl` `switch d.Mod` gains a `FuncDerive` case
  that returns after `deriveDecl`, so the existing gensym-scope setup (for Result/
  Option functions) is bypassed for a derive — correct, since a derive func is not a
  Result/Option function and emits its own complete body.

No CRITICAL/MAJOR findings.

## Assumptions

- `backend.Transpile` already calls `sema.Resolve` (confirmed in lower.go doc + US-033
  notes), so `e.info.Structs`/`FromRegistry` are populated when `deriveDecl` runs.
- Override values render via `exprText` (a fresh sub-emitter), avoiding a Builder copy.
