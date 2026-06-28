# Plan Coverage Audit — US-003

- FR-1 (run both over whole corpus) → `TestSemaLegacyParity` iterates manifest
  `KindCheck` cases, runs both checkers, fails on zero cases. Covered.
- FR-2 (compare by key) → `finding` tuple (file, line, feature, code, severity).
  Covered.
- FR-3 (identical except documented allowlist; fail on undocumented + stale) →
  `knownDivergences` allowlist + symmetric-difference diff with stale detection.
  Covered.
- FR-4 (document divergences; markers reflect sema) → DECISIONS.md section;
  markers already aligned. Covered.
- No scope creep: only one new test file + one doc append.

Verdict: full coverage, no gaps.
