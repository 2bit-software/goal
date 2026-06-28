# Tasks Audit — US-004

- T1 has no dependencies and is the foundation; it leaves the tree compiling
  (single-file edit, all referenced symbols exist).
- T2 depends on T1 (test exercises the rewired path).
- T3 depends on T1+T2 (verifyCommands).
- Each task touches <=2 files. Ordering is a valid topological sort.
- Every acceptance criterion maps: AC1 (no check import) -> T1; AC2 (dedup
  preserved) -> T1 + existing tests; AC3 (e2e corpus unchanged) -> T2.

No CRITICAL/MAJOR findings.

## Assumptions
- None beyond those already recorded in the plan audits.
