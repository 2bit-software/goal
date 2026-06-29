# Scaffold notes — SEAM-005

In-place idiomatization (signature change) — the refactor's side-by-side model
does not apply: one exported function cannot have two coexisting versions, and
open-E Result lowers to the identical Go `(T, error)`, so old and new emit are the
same binary contract.

## Edits (selfhost/ only; internal/ Go stays the lowered (T,error) mirror)

- `selfhost/sema/package.goal` — `AnalyzePackageInDir` → `Result[[][]Diagnostic, error]`
  (explicit unpack of the 3-value `…With`, then `Result.Ok`/`Result.Err`).
- `selfhost/typecheck/typecheck.goal` — `Load` → `Result[*Package, error]`
  (wrapping guards become `return Result.Err(fmt.Errorf(...))`; final `Result.Ok(p)`).
- `selfhost/typecheck/checker.goal` — `TypeChecker.Check` interface method +
  `GoTypesChecker.Check` → `Result[[]Diagnostic, error]`; body `p := Load(pkg)?`,
  `return Result.Ok(diags)`.

## Why no internal/ or oracle-test edits

- Open-E `Result[T, error]` lowers to native Go `(T, error)`; an interface method
  returning Result lowers to a `(T, error)` method. So the transpiled selfhost Go
  signatures are byte-identical to before.
- `internal/typecheck/*.go`, `internal/sema/*.go` are the Go bootstrap compiler;
  Go cannot express Result, and they already ARE the lowered `(T, error)` mirror.
- Port-gated oracle tests (`internal/typecheck/checker_test.go`,
  `internal/sema/package_test.go`) call these two-value (`v, err := …`) and assert
  `var _ TypeChecker = GoTypesChecker{}`; they compile against both the internal Go
  AND the transpiled selfhost Go (which lowers to `(T, error)`), so they stay green
  unchanged — that IS the seam staying swappable.

## Verification

- `task check`, `task build`, `task fixpoint` all green.
- Corpus behavioral tier unchanged (covered by `task check`).
