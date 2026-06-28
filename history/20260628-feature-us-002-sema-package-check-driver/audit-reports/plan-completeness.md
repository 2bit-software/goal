# Plan Audit — Coverage (US-002)

Every spec element maps to a plan element:

- FR-1 (parse/resolve/enrich/check) -> package.go `AnalyzePackageInDirWith` steps 1-5.
- FR-2 (per-file, input order) -> step 5 loop + `TestAnalyzePackageInDirCrossFileExhaustiveness`.
- FR-3 (cross-file + foreign finding) -> `TestAnalyzePackageInDirForeignEnrichedDeriveFinding`.
- FR-4 (non-fatal foreign errs, injectable resolver) -> `...With` signature + fake resolver.
- AC `task check`/`task build` -> verify step.

No plan element lacks a requirement (no scope creep). No CRITICAL/MAJOR findings.

## Assumptions
- Two exported entry points (plain + `...With`) mirror legacy check.go; only the
  plain one is strictly required by the AC, the `...With` variant supports tests
  and the later LSP consumer.
