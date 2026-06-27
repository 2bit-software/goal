# Audit — Completeness

Scope: small, self-contained corpus-hardening story (relocate two inline package
sources into on-disk fixtures + index + runner). No external dependencies.

## Findings

- MINOR: "verifies it passes" (FR-4) leaves the pass bar implicit. Resolved by
  the AC + error-handling section: transpile succeeds, every generated file is
  valid Go, and the package compiles. The cross-file fixture historically
  compiled; the foreign fixture's compile is a new (stronger) bar — covered by
  the risk note (fall back to transpile-success + valid-Go if the derive body is
  not self-contained).
- MINOR: counts (51/50/4) are asserted to be unchanged; the new package count is
  not pinned in the spec. The generate-count test will pin it.

No CRITICAL or MAJOR findings. Happy path, empty-import-map, zero-case guard, and
failure paths are all specified.

## Assumptions

- Package cases are `Kind=transpile` + `Mode=package` (not a new Kind).
- Foreign packages resolve in-module via the existing `DefaultResolver`; no
  external module fetch is needed.
- Removing the now-redundant inline Go tests is optional and not required to
  pass the story.
