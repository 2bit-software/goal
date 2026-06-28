# AI-Consumer Readiness Audit — US-003

## Findings

- All seams are concrete and named: `corpus.Load`, `manifestPath`, `repoRoot`,
  `KindCheck`, `check.Analyze`, `SemaCheck`, `check.OffsetToPosition`,
  `check.Diagnostic`, `Severity.String()`. An implementer needs no guessing.
- The comparison key (file, line, feature, code, severity) is fully specified and
  testable.
- The four allowlist entries are enumerated with exact file/line/feature/code/
  severity values in research-findings.md, so the allowlist is writable verbatim.
- Acceptance criteria map one-to-one to test assertions.

## Assumptions surfaced

- The gate lives in `internal/corpus` (alongside the existing check/sema runners)
  and reuses the committed manifest rather than walking `testdata/check/**`
  directly — consistent with every other corpus runner.
- Comparison is single-file (no package/foreign enrichment) so both checkers see
  the same facts: `check.Analyze` and `SemaCheck` are both single-file entry
  points.
- `// want` markers for the three derive-convert improvements already contain the
  sema Error substrings, so AC's "markers reflect sema behavior" is already met
  without editing the fixtures.

## Verdict

Implementable without clarification.
