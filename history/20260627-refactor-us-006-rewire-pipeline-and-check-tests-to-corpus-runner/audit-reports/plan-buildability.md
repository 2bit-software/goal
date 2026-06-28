# Plan Audit: Buildability — US-006

## Checks

- Dependency order: valid. The corpus runners, adapters, and manifest already
  exist and compile; the plan only adds external-package test files that call them.
- Interface contracts agree: `corpus.RunTranspile/RunDoctest(root, Case, Transpiler)`
  and `corpus.RunCheck(root, Case, Checker)` signatures match the planned calls;
  `TranspilerFunc(pipeline.Transpile)` and `CheckerFunc(check.Analyze)` are the
  documented adapters.
- File paths verified: both files exist at the stated paths; repo-root depth `../..`
  matches the corpus package's own usage.
- Import cycle correctly addressed: external `_test` packages break the
  corpus->pipeline / corpus->check cycle. This is the one real buildability risk and
  the plan handles it explicitly.

No CRITICAL, no MAJOR findings.

### MINOR-1: Coexisting package decls in one directory
`internal/check` will then contain both `package check` test files and a
`package check_test` file. This is legal Go. The final `go test` gate confirms it.

## Assumptions
- No symbol collisions between the new external-package consts (`manifestPath`,
  `repoRoot`) and existing internal-package symbols — they live in different
  packages, so even identical names cannot collide.
