# Tasks Audit — US-009

Coverage: every plan file appears in a task; every FR/AC covered.
- Task 1 -> FR-1/AC1 (12 .goal files).
- Task 2 -> FR-2/FR-3/AC2/AC3 (both gates).
- Task 3 -> closeout + project gates (task check/build/fixpoint).

Ordering valid (1 -> 2 -> 3), each independently committable, each <=5 files
(except Task 1's 12 verbatim copies, which are a single atomic mechanical copy
and compile together — acceptable as one unit). Concrete verify commands given.

No CRITICAL/MAJOR findings.

## Assumptions
- Task 1's 12-file copy is treated as one atomic task despite the >5-file
  guidance, because the files are a verbatim package copy that only compiles as
  a set (matches how US-007/US-008 ported multi-file packages in one step).
- Behavioral test-file set is finalized empirically in Task 2.
