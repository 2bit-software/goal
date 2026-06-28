# Plan Buildability Audit — US-003

- Dependency order valid: DECISIONS.md doc first, then the test that cites it. No
  forward references.
- All referenced symbols exist and were verified in the codebase: `Load`,
  `manifestPath`, `repoRoot`, `KindCheck`, `check.Analyze`, `SemaCheck`,
  `check.OffsetToPosition`, `check.Diagnostic`, `Severity.String()`.
- File path `internal/corpus/parity_test.go` does not exist yet (no conflict).
- Allowlist values are concrete (file/line/feature/code/severity enumerated in
  research-findings.md), so no placeholders.
- Single Go test file compiles against the existing package with no new deps.

Verdict: buildable as specified.
