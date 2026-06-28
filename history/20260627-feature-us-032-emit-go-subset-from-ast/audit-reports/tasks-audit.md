# Tasks Audit — US-032

## Coverage
- FR-1 → Task 1 + Task 3; FR-2 → unchanged engine (no task needed, correct);
  FR-3 → Task 2 + Task 3; FR-4 → Task 1. All requirements covered.
- Plan file inventory all present: emit.go → Task 1, plain_full.goal → Task 2,
  backend_test.go → Task 3. No file outside the plan referenced (no scope creep).

## Ordering
- Valid DAG: Task 1 (none), Task 2 (none), Task 3 (depends on 1+2). No cycles.
- Codebase compiles after Task 1 (emitter addition is additive), after Task 2
  (fixture only), and after Task 3 (tests). Each task independently committable.

## Executability
- Each task names concrete files, real AST field names, and a runnable verify
  command. Task 1 references the existing `ifStmt`/`forStmt`/`block` helpers as
  the pattern to mirror. Task 3 reuses the established `corpus.RunCompile` seam.
- Each task touches a single file (≤5). No vague directives.

## Sizing
- Three appropriately-sized tasks; none trivial, none oversized. All completable
  in one agent turn.

## Findings
- No CRITICAL, MAJOR, or MINOR findings. Tasks are ready for implementation.

## Assumptions
- Task 1 dispatches `caseClause` from `switchStmt` rather than through the generic
  `block`/`stmt` path, because a `*ast.CaseClause` is a clause, not a normal
  statement line. This matches go/printer's handling.
- The full-subset fixture is stdlib-only so the corpus temp-module build resolves
  offline (consistent with US-007/US-026 behavioral fixtures).
