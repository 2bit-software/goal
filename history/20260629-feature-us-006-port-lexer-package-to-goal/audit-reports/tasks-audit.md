# Tasks Audit — US-006

## Checks
- Coverage: every plan file appears in a task (lexer.goal -> T1; selfhost.go +
  port_test.go -> T2/T3). All acceptance criteria mapped.
- Ordering: T1 and T2 are independent; T3 depends on both; T4 verifies. Valid
  topological order.
- Sizing: each task touches <=2 files and is a single-turn unit. None trivial,
  none oversized.
- Each task has a concrete verify command.

## Findings
No CRITICAL or MAJOR findings.

## Assumptions
- BuildAndTest is extended (signature gains a deps param) rather than duplicated;
  the one existing caller (token test) is updated in the same task to pass nil.
