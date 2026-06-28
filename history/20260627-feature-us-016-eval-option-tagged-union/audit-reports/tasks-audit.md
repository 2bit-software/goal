# Tasks Audit

- Coverage: every spec AC maps to a task (Task 2 construction, Task 3 unwrap,
  Task 4 tests). Every plan file appears (value.go T1, eval.go T2, interp.go T3,
  option_test.go T4).
- Ordering: constants (T1) before use (T2/T3); tests (T4) after impl; verify (T5)
  last. No forward references.
- No scope creep: no task touches a file outside the plan inventory.

No CRITICAL/MAJOR findings. Tasks are executable as ordered.
