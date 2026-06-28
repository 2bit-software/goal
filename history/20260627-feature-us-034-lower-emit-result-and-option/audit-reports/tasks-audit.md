# Tasks Audit — US-034

## Coverage

- Every plan file appears in a task: `lower.go` (T1), `emit.go` (T2),
  `backend_test.go` (T3), `prd.json`/`progress.txt` (T4). ✓
- Every FR maps to a task: FR-1..FR-6 in T1/T2; AC in T3. ✓

## Ordering & size

- Dependency order is a valid topological sort: T1 (helpers, no deps) -> T2
  (wiring) -> T3 (test) -> T4 (gate). ✓
- Each task touches <= 1-2 files; each is independently committable and
  single-turn. ✓
- Each task has a concrete verify command. ✓

## Findings

No CRITICAL or MAJOR. MINOR: T2 is the largest task (one file, many small edits)
but stays within one file and one cohesive concern (Result/Option lowering in the
emitter), so it remains single-turn. Recommend PASS.

## Assumptions

- The 03-result and 04-option example inputs are stable and enumerable from the
  manifest / examples dirs.
