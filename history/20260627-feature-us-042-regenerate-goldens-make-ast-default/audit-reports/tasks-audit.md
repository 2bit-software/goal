# Tasks Audit — US-042

## Coverage
- FR-1 -> Task 4; FR-2 -> Task 1/2; FR-3 -> Task 3; FR-4 -> Task 4; FR-5 -> Task 4.
- Every plan file appears in a task. prd.json/progress.txt are loop bookkeeping
  (handled at finalize), not implementation tasks — intentionally omitted.
- No scope creep: no task touches files outside the plan inventory.

## Ordering
- DAG valid: 1 -> 2 -> 3; 4 independent; 5 depends on 3 & 4. No cycles.
- Compiles after each task: Task 1 adds a flag-gated test (compiles); Task 2 only
  changes golden text (no Go change); Task 3 swaps a function value (type-compatible);
  Task 4 is independent and self-consistent.

## Executability
- Each task names concrete files (<=5 each), a concrete verify command, and reuses
  named patterns (Transpiler seam, -update-snapshots, isDoctestSidecar).

## Sizing
- All tasks single-turn sized; none trivial (Task 2 regenerates ~55 goldens).

No CRITICAL/MAJOR findings.

## Assumptions
- prd.json/progress.txt updates happen in the loop finalize step, not as plan tasks.
