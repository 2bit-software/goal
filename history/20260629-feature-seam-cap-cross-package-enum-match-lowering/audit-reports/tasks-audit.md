# Tasks Audit

## Findings
No CRITICAL or MAJOR findings.

- Ordering is valid: Task 1 (sema) and Task 2 (matchQualifier) are independent
  foundations; Task 3 (fixtures/tests) depends on both; Task 4 (selfhost mirror)
  is independent but grouped after for clarity; Task 5 verifies.
- Coverage: every plan file appears in a task; every FR (FR-1..FR-4) is covered.
- No task touches >5 files.

### MINOR
- Task 3 bundles fixture + test + manifest (4 files) — within the <=5 limit, kept
  together because they are mutually dependent (the test/corpus need the fixtures).

## Assumptions
- selfhost mirror edits will not be behaviorally exercised by fixpoint (selfhost
  source has no cross-package enum match yet) but must remain valid goal so both
  bootstrap stages still emit byte-identical output.
