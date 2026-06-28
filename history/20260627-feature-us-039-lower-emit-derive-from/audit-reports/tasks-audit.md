# Tasks Audit

- Coverage: every FR (1-6) and all acceptance criteria map to Tasks 2/3; every
  plan file (lower.go, emit.go, backend_test.go, prd.json, progress.txt) appears.
- Ordering: valid DAG — Task1 (pure helpers) -> Task2 (emitter, uses helpers) ->
  Task3 (tests) -> Task4 (gate+flip). Codebase compiles after Task2 (derive path
  emits; no caller breakage) and after each subsequent task.
- Executability: each task has a concrete verify command and references specific
  ASTs/functions. Each touches <= 3 files (Tasks 1-3 touch 1 file; Task 4 touches 2).
- Sizing: appropriate; no trivial tasks.

No CRITICAL/MAJOR findings.

## Assumptions
- Tasks 1 and 2 may be implemented together in one turn (same review boundary) since
  Task 1's helpers are only consumed by Task 2 — does not affect compilation order.
