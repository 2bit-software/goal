# Tasks Audit — US-003

## Coverage
- FR-1/FR-2/FR-3 → Task 2. FR-4 → Task 1. All gates → Task 3.
- Both plan files (`DECISIONS.md`, `internal/corpus/parity_test.go`) appear in
  tasks. No scope creep.

## Ordering
- Task 1 (doc) → Task 2 (test cites doc) → Task 3 (gates). Valid, no forward refs.

## Executability
- Each task has a concrete verify command. Granularity is appropriate for a
  one-test-file story.

Verdict: tasks complete, ordered, executable. Ready to implement.
