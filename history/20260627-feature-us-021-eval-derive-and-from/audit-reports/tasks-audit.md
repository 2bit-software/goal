# Tasks Audit — US-021

## Coverage
- FR-1..FR-5 all map to Task 2 (derive evaluation); Task 4 tests all acceptance
  criteria. Every plan file (interp.go, derive.go, eval.go, derive_test.go)
  appears in a task. No scope creep.

## Ordering
- Valid DAG: T1 (register) -> T2 (evaluate) -> T3 (intercept) -> T4 (test).
- The tree compiles after each task: T1 adds an unused-but-valid map field +
  routing; T2 adds a new file with self-contained functions (evalDerive unused
  until T3 — Go permits unused methods, only unused locals/imports fail); T3 wires
  the call; T4 adds tests.

## Executability
- Each task names exact files, functions, and the existing patterns to mirror
  (callConversion, defaults_test.go). Verify commands are concrete.
- No task exceeds 5 files (max 1 file each).

## Sizing
- All tasks are substantive and fit a single turn. Task 2 is the largest but is
  one cohesive new file.

## Result
No CRITICAL/MAJOR findings. Recommend pass.

## Assumptions
- Practically, Tasks 1-4 are committed together as one focused change (the loop
  commits the whole story atomically); the ordering still governs build-up.
