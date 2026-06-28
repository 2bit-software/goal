# Tasks Audit — US-020

- Coverage: every plan file (`goal_decl.go`, `parser.go`, `goal_decl_test.go`)
  appears in a task; every FR maps to Task 1/2 and is tested in Task 3.
- Ordering: Task 1 has no deps; Task 2 depends on 1; Task 3 on 1+2; Task 4 on
  all. Valid topological order, no forward references.
- No task touches more than 3 files. No CRITICAL/MAJOR findings.

## Assumptions

- Tests live in the internal `parser` package (same as `parser_test.go`).
- Example `.goal` inputs are read from `../../features/...`.
