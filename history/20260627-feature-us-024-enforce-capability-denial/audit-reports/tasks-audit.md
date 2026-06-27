# Tasks Audit

## Findings

No CRITICAL findings.
No MAJOR findings.

### Coverage
- FR-1..FR-5 covered by Task 1; all acceptance criteria asserted by Task 2.
- Plan files all appear: interp.go + host.go (Task 1), cap_deny_test.go (Task 2).
- No scope creep — no files outside the plan are referenced.

### Ordering
- Task 1 (none) -> Task 2 (depends on Task 1): valid DAG, no cycles.
- Task 1's two-file edit pair leaves the package compiling (signature + sole
  caller changed together); Task 2 then compiles against it. Verify step is
  `go build ./...` after Task 1, `go test ./internal/interp/...` after Task 2.

### Executability
- Each task has concrete instructions with actual code shapes and a runnable
  verify command. Task 1 touches 2 files, Task 2 touches 1 — within the 5-file
  limit.

### Sizing
- Both tasks are single-turn sized and non-trivial. Could be merged into one
  turn, but the split (production then tests) is clean.

## Assumptions
- Tests use the same construction style as the existing `cap_io_test.go`
  (parser + sema.Resolve + New options), since that is the established pattern.
