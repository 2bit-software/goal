# Tasks Audit — US-002

- Three tasks, each independently committable, ≤3 files, concrete verify command.
- Ordering respects the dependency graph: renames -> harness -> tests.
- Coverage: FR-1 (T2,T3), FR-2 (T2,T3), FR-3 (T1,T3), FR-4 (T2). Every plan file
  appears in a task. No orphan tasks, no scope creep.
- No CRITICAL/MAJOR findings.

## Assumptions
- The negative-case sample (`func f() int { return }`) transpiles through the
  front-end (sema does not hard-reject it) but fails `go build`. If sema rejects it
  at transpile time, BuildTranspiled still returns a (different) error, so the test
  assertion "error is non-nil" holds either way.
