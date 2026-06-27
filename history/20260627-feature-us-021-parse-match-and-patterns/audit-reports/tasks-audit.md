# Tasks Audit

## Coverage
- FR-1..FR-5 covered by Task 1; all acceptance criteria covered by Task 2;
  verify gate by Task 3.
- Every plan file (`goal_match.go`, `parser.go`, `goal_match_test.go`) appears in
  a task. No scope creep.

## Ordering
- Task 1 → Task 2 → Task 3 is a valid linear DAG.
- After Task 1 the tree compiles (dispatch + methods added together). After Task
  2 tests exist and pass. Task 3 runs full gates.

## Executability
- Each task touches ≤ 2 files, completable in one turn, with a concrete verify
  command.

No CRITICAL/MAJOR findings.

## Assumptions
- Test helper `readExample` is reused from the same package (`goal_decl_test.go`).
