# Plan Buildability Audit — US-004

- Dependency order is valid: fix the fixer, rebuild, run it, verify gates, commit.
  No forward references.
- Interface contracts use real signatures grounded in the existing
  `internal/fix` code (ast/sema/textedit/token types already imported there).
- File paths verified against the codebase (`internal/fix/resultsig.go`,
  `selfhost/`, `prd.json`, `progress.txt` all exist).
- Integration points are specific: `fix.File`'s loop already calls
  `fixResultSig`; only its internals change; `fixPropagate`/`reportCallSites`
  untouched.

No CRITICAL or MAJOR findings.

## Assumptions
- goal's `ast.Walk` + `visitFn` adapter (already used in `optionDerefRewrites`)
  is sufficient to scan identifier references for `unsafeRefOffset`.
- `textedit.ZeroLit(successT, decls, 0)` is the correct zero-value oracle for
  `validPropagationReturn` on candidate enclosing functions (mirrors existing
  usage in `classifyReturns`).
