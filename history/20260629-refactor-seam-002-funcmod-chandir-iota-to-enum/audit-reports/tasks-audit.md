# Tasks Audit — SEAM-002

Audited tasks.md against implementation-plan.md and business-spec.md.

## Coverage — PASS
- FR-1 (FuncMod enum): Task 1.
- FR-2 (ChanDir enum): Task 1.
- FR-3 (consumers -> match): Task 1 (ast.goal:170), Task 3 (sema), Task 4 (backend).
- FR-4 (qualified construction + zero-value fix): Task 2.
- FR-5 (token.Kind iota refusal documented): Task 6.
- FR-6 (behavior + zero-value invariant): Task 2 (construction) + Task 6 (gates).
- Every plan file-inventory entry appears in a task (selfhost/ast, parser, sema,
  backend; internal/ast/ast_test.go + new funcmod_test.go; DECISIONS.md, prd.json,
  progress.txt). No file outside the plan is referenced (no scope creep).
- Every AC maps to a task verify step; the two DECISIONS.md doc ACs are Task 6.

## Ordering — PASS (valid DAG)
Task 1 (foundation, no deps) -> Tasks 2,3,4,5 (depend on 1; 5 is test-only) ->
Task 6 (depends on 1-5). No circular or forward dependency.

## Executability — PASS
Each task lists concrete files and per-site target forms with line numbers and
exact before/after spellings (taken from the buildability-audited plan). Each has
a verify note; the real gates run in Task 6.

## Sizing — PASS
Each task touches 1-3 files. None trivial, none oversized.

## CRITICAL / MAJOR
None.

## MINOR / Accepted deviations
- M-1 (accepted, by design): the generic task-quality rule "codebase compiles
  after each task" is INTENTIONALLY not satisfied between Tasks 1-5. This is an
  ATOMIC SEAM: once the iota constants become enum variants, any not-yet-converted
  `==`/`switch` is a §9/undefined-identifier compile error, so the tree is red
  until the last edit lands. tasks.md states this explicitly and defers all gating
  to Task 6 (one commit). Splitting to keep each step green is impossible for a
  sealed-type conversion and is the whole point of the SEAM relaxed gate
  (DECISIONS.md "Seam methodology"). Not a defect.
- M-2 (informational): Task 5 (test relocation) is technically green on its own
  against Go-iota internal/ast, but is committed together with Tasks 1-4 so the
  port gate (which compiles ast_test.go against the enum-transpiled selfhost/ast)
  stays green in the same commit.

## Assumptions
1. The PostToolUse `task check` hook firing red between edits is expected and not
   a stop condition (documented project pattern in progress.txt).
2. A single atomic commit is the correct granularity for a SEAM story (per the
   PRD description and DECISIONS.md Seam methodology), overriding the workflow's
   default "independently committable task" guidance.
3. No new selfhost test fixture is required; the existing port gates + fixpoint
   exercise every converted form (confirmed by the plan-buildability audit).

## Recommendation
PASS — no CRITICAL/MAJOR; the one structural deviation (atomic, not
step-wise-green) is inherent to sealed-type conversion and is documented.
