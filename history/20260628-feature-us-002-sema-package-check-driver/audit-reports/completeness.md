# Audit — Completeness (US-002)

Scope is a single internal Go API mirroring an existing legacy function
(check.AnalyzePackageInDir). The spec covers happy path (FR-1/FR-2), the
cross-file + foreign case (FR-3), and the error paths (parse error fatal,
foreign-resolution error non-fatal, FR-4).

## Findings

- MINOR: Empty input (`srcs == nil`) is unspecified. Resolution: returns an
  empty `[][]Diagnostic` (len 0) and nil error — falls out naturally from the
  per-file loop; no explicit handling needed.
- MINOR: Order of import aggregation across files when two files import the same
  path with different aliases is unspecified. Resolution: EnrichForeign dedupes
  by import path (foreign.go `loaded` map), so the first-seen alias wins; matches
  legacy behavior closely enough for the corpus. Not exercised by this story.

No CRITICAL or MAJOR findings.

## Assumptions

- The convenience entry point discards the foreign-error slice (mirrors legacy
  AnalyzePackageInDir); a `...With` variant surfaces it for tests/LSP.
- Imports are aggregated across ALL package files before a single EnrichForeign
  call (the merged Info is shared by every file's Check).
