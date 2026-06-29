# Scope — SEAM-005

## What is being refactored and why

The self-hosted compiler's fallible exported/interface API still returns Go
`(T, error)` tuples. SEAM-005 lifts the cluster that is **pure-propagation and
blocked ONLY by cross-package callers** to goal `Result[T, E]` + `?`, so error
handling at the compiler's seams reads idiomatically. Result/Option already work
cross-package (special-cased), so this story does NOT depend on the enum/sealed
capability gaps.

## CONVERT (selfhost/ — scope-blocked-only, pure propagation)

- `selfhost/typecheck/typecheck.goal` — `Load(pkg) (*Package, error)` →
  `Result[*Package, error]`. Internal sites WRAP context (`fmt.Errorf("…: %w")`),
  so they stay statement-form `if err != nil { return Result.Err(fmt.Errorf(...)) }`
  (no `?` inside Load — `?` would drop the wrap); final `return Result.Ok(p)`.
- `selfhost/typecheck/checker.goal` — `TypeChecker.Check` interface method +
  `GoTypesChecker.Check` → `Result[[]Diagnostic, error]`. Body uses the real
  idiom gain: `p := Load(pkg)?` then `return Result.Ok(diags)`.
- `selfhost/sema/package.goal` — `AnalyzePackageInDir(srcs, dir) ([][]Diagnostic, error)`
  → `Result[[][]Diagnostic, error]`. Delegates to the 3-value `…With`, so it
  unpacks then `return Result.Ok(diags)` / `return Result.Err(err)`.

## CARVE OUT (semantic non-fits, KEPT + documented — not scope)

- `parser.ParseFile (*ast.File, error)` — returns a **partial AST AND joined
  errors simultaneously**; `project.PackageClause` and `pipeline.declSites`
  consume the partial file on error (documented tolerance). A `Result` Err arm
  carries no file → behavior change. Value-AND-error contract; keep.
- `sema.EnrichForeign (…) []error` — error ACCUMULATOR (appends + continues); no
  `?` site.
- `sema.AnalyzePackageInDirWith (…) ([][]Diagnostic, []error, error)` — 3-value;
  the per-import error slice is meaningful alongside success.
- `sema.foreignDecls` / `sema.goalForeignDecls` — 4-value multi-return.
- `sema.moduleResolve` / `sema.readModulePath` / `sema.constIntLit` — comma-ok
  control flow (two distinct failures collapsed to one bool).

## What must NOT change

- Emitted Go SIGNATURE of the converted fns is unchanged (open-E Result lowers to
  native `(T, error)`; interface method lowers to `(T, error)`), so:
  - `internal/` Go bootstrap mirror stays plain `(T, error)` (Go cannot express
    Result) and remains the behavioral mirror.
  - Shared/port-gated oracle tests (`internal/typecheck/checker_test.go`,
    `internal/sema/package_test.go`) keep their two-value `v, err := …` call
    sites and `var _ TypeChecker = GoTypesChecker{}` — they compile against BOTH
    the internal Go and the transpiled selfhost Go.
- `task fixpoint` stays green (stage1==stage2 on the new Result/? source).
- Corpus behavioral tier unchanged.
