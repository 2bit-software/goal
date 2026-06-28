# Tasks — US-004

## T1 — Rewire checkPackage onto sema (foundation, no deps)
- In `cmd/goal/main.go`:
  - Change `checkDiag.severity` field type from `check.Severity` to `sema.Severity`.
  - In `cmdCheck`, change `d.severity == check.Error` to `d.severity == sema.Error`.
  - In `checkPackage`, swap `check.AnalyzePackageInDir` -> `sema.AnalyzePackageInDir`.
  - Drop `check.OffsetToPosition`; build positions from `d.Pos.Line` / `d.Pos.Col`.
  - Convert depth severity with `sema.Severity(d.Severity)`.
  - Update the `checkPackage` doc comment (lexical stage is now AST/sema).
  - Remove the `"goal/internal/check"` import.

## T2 — End-to-end corpus test (depends on T1)
- In `cmd/goal/main_test.go`, add `TestCheckCorpusOutputUnchanged`: copy a
  corpus KindCheck case (testdata/check/02-match/non_exhaustive_stmt.goal) into
  a goal module temp dir, run `goal check`, and assert the rendered output
  contains the `[non-exhaustive-match]` error line and that check exits non-zero.

## T3 — Verify (depends on T1, T2)
- Run `task check` and `task build`. Fix any failures, re-run until green.
