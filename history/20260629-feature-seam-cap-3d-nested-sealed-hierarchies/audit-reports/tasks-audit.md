# Tasks audit

- Coverage: all FRs and plan files covered across 3 tasks. No scope creep.
- Ordering: valid DAG (sema → backend → tests/docs). Each task compiles
  independently (cascade is additive; sealedEmbeds is new; tests last).
- Executability: each task names files, concrete helper signatures, reference
  patterns (sealed_match_test.go), and a verify command.
- Sizing: each task ≤ 4 files. Reasonable for one turn.
- Note (MINOR): the PostToolUse `task check` hook reports transient failures
  between edits in a multi-file mirror; only the final green state matters
  (documented codebase pattern). Not blocking.

No CRITICAL/MAJOR findings.

## Assumptions
- selfhost .goal mirrors must land in the same task as their internal/ twin to
  keep the port gate compiling at task boundaries.
- Tests/docs deferred to Task 3 once the capability exists.
