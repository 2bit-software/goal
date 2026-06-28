# Plan Buildability Audit — US-004

- Dependency order valid: edit main.go, then add test, then verify. No forward
  references.
- Interface contracts agree: `sema.AnalyzePackageInDir` returns
  `[][]sema.Diagnostic`; `sema.Diagnostic.Pos` is `token.Pos` with `.Line`/`.Col`
  (verified in internal/token/token.go and internal/sema/check.go).
- Severity conversion `sema.Severity(typecheckDiag.Severity)` is valid: both are
  `int`-based with Error=0/Warning=1 (verified in check.go line 51-55 and
  sema/check.go line 25-32); the conversion does not name `check.Severity`, so
  the check import can be dropped.
- File paths verified: cmd/goal/main.go, internal/sema/package.go,
  testdata/check/02-match/non_exhaustive_stmt.goal all exist.

No CRITICAL/MAJOR findings.

## Assumptions
- `goal check <single .goal file>` accepts a file path (confirmed: existing
  TestCheckDepthNoteOmitsGeneratedDump passes a file path to `run`).
