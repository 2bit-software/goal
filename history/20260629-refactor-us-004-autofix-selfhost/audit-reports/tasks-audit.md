# Tasks Audit — US-004

## Coverage
- FR-1/AC-1 -> Task 2; FR-2/AC-6 -> Task 1; FR-3/AC-2 -> Task 2;
  FR-4/AC-3..5 -> Task 3. All requirements covered.
- File inventory covered: `internal/fix/resultsig.go` (Task 1),
  `selfhost/**/*.goal` (Task 2). `prd.json`/`progress.txt` updates are the
  loop-runner finalize step (post-verify), not implementation tasks.

## Ordering
- Valid DAG: Task 1 -> Task 2 -> Task 3. Codebase compiles after Task 1
  (fixer change is self-contained), after Task 2 (selfhost still compiles via
  trusted toolchain), and after Task 3 (gates green).

## Executability
- Each task has concrete instructions referencing specific functions and exact
  Skip messages, and a runnable verify command. No task exceeds 5 files.

## Sizing
- Task 1 is the substantive change; Tasks 2-3 are run-and-verify. All single-turn.

No CRITICAL or MAJOR findings.

## Assumptions
- `prd.json` / `progress.txt` updates are handled by the loop finalize step, so
  they are intentionally not separate implementation tasks.
