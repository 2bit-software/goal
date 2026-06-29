# Scope — US-011 idiomatic audit: backend

## What is being refactored and why

`selfhost/backend` (arity/backend/doctest/emit/lower/package.goal, ~3,447 LOC)
is the largest ported package. The US-001 port + US-004 autofix left manual
`if err != nil { return zero, err }` propagation in place because `goal fix`
only auto-converts propagation sites whose enclosing function is already
`Result`-returning (it emits `suggestion`/`skipped` reports instead). The
audit replaces genuine pure-propagation sites with `?` and converts one clean
leaf helper to a `Result` signature, so the emission code reads as idiomatic
goal.

## Key finding (front-end rule)

`selfhost/sema/sema.goal:39`: a `?` host may be "a Result, a `func(…) error`,
or a tuple ending in error". So `?` is legal inside a plain `(T, error)`
function, and `f()?` lowers to exactly `v, err := f(); if err != nil { return
zero, err }` — byte-identical to the manual block it replaces. This lets the
audit `?`-ify pure-propagation sites without changing any signature.

## What the new code should look like

Behavior-preserving conversions only:

1. `deriveBody` (emit.goal): `([]string, error)` -> `Result[[]string, error]`
   (open-E, lowers to identical Go). Returns become `Result.Ok/Err`; its sole
   caller (resolveField) becomes `?`. This is the concrete "fallible helper
   uses Result" example (same shape as US-009 sema.Analyze).
2. `?` propagation at PURE sites (no signature change, byte-identical):
   - resolveField: 4× `elemConv(...)?`, 1× `e.deriveBody(...)?`.
   - Emit (backend.goal): `emitFile(...)?`.
   - Transpile (backend.goal): `goBackend{}.Emit(...)?`.

## What must NOT change (preserved)

- Public/oracle-pinned signatures: `Transpile`, `TranspilePackage`, the
  `Backend.Emit` / `Formatter.Format` interface methods, `GoFormatter.Format`,
  `goBackend.Emit` — kept `(T, error)` (they still gain `?` at pure sites).
- WRAPPING propagation sites (error context is behavior-load-bearing): keep
  manual — `parse:`/`doctests:`/`generated Go did not parse:` (backend.goal),
  `nested field %q: %w` (deriveBody->resolveField), genConversion's
  `e.fail(...)` conversion of resolveField's error, and package.goal's
  `format ...` wraps. `?` propagates the error UNCHANGED, so using it here
  would drop the context = behavior change.
- `resolveField` signature stays `([]string, error)`: its callers genConversion
  (e.fail) and deriveBody (`nested field` wrap) both transform the error, so it
  cannot become `Result` without `match` (out of behavior-preserving scope).
- No switch->match: backend declares NO in-file `enum`; every switch is over a
  non-enum scrutinee (ast category interfaces — unsealable per US-007 §9 — or
  token.Kind / rune / bool). Documented refusal.

## Verification

`goal fix selfhost/backend/*.goal` (no remaining auto-convertible sites);
`go test ./internal/selfhost -run TestPortedBackendPackage`; `task check`;
`task build`; `task fixpoint` (FIXPOINT OK).
