# AI-Consumer Readiness Audit — US-004

## Findings

- The spec plus technical-requirements-research.md name every symbol involved
  (`sema.AnalyzePackageInDir`, `sema.Diagnostic.{Pos,Severity,Feature}`,
  `checkDiag`, `dedupKey`, `runDepthChecks`) and the exact severity-conversion
  approach. An implementer can proceed without clarifying questions.
- Acceptance criteria are test-assertable: `ok` on clean, non-zero on violation,
  "depth stage unavailable" note without "--- generated ---", and dedup of the
  lexical misfire — all already covered by existing cmd/goal e2e tests.

No CRITICAL or MAJOR findings.

## Assumptions

- `sema.Severity(d.Severity)` (numeric conversion) is acceptable for mapping a
  depth finding's `check.Severity` into `checkDiag.severity` without importing
  `internal/check`.
