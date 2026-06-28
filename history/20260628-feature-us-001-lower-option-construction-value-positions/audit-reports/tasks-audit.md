# Tasks Audit

## Findings

No CRITICAL findings. No MAJOR findings.

### Coverage
- FR-1/FR-2/FR-3 covered by T1 (classifier/encoding) + T2 (interception) + T3
  (helper injection). FR-4 guarded by T5. Test criterion by T4.
- Every plan file (lower.go, emit.go, package.go, backend_test.go) appears in a task.
- No scope creep — no files outside the plan inventory.

### Ordering
- Valid DAG: T1 -> T2 -> T3 -> T4 -> T5. Each task compiles after the prior: T1 adds
  unused-by-nothing leaf helpers (Go allows unused package funcs); T2 wires them; T3
  injects; T4 tests; T5 verifies.

### Executability
- Each task names concrete files, functions, and existing patterns to mirror
  (`needsFmtImport`, `resultPrelude`, `needsResultPrelude`). Each has a verify step;
  none touches >5 files (max 2).

### Sizing
- Tasks are single-turn sized; none trivial, none oversized.

## Assumptions
- T1's leaf helpers compiling "unused" between T1 and T2 is fine because Go does not
  error on unused package-level functions (only unused locals/imports).
- The boxed-temporary helper name `goalSome` is internal; tests assert behavior, not
  the name.
