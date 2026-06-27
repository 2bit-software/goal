# Plan Coverage Audit — US-004

## Findings

No CRITICAL or MAJOR findings. Every spec element traces to a plan element:

- FR-1 (construct from AST+sema) → `New(*ast.File, *sema.Info)` in interp.go.
- FR-2 (run main) → `Run()` scanning file.Decls for `func main`.
- FR-3 (missing main → named error) → `Run()` error path; test
  `TestRunMissingMainErrors`.
- AC trivial-program → `TestRunTrivialMain`.

No scope creep: only interp.go + its test are added; no source files modified.

## Assumptions

- Single-file `*ast.File` input (package-mode deferred), matching the AC program.
- Empty `main` body is a no-op; statement eval is US-005+.
