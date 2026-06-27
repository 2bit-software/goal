# Tasks Audit (US-025)

## Coverage
- FR-1 → Task 2; FR-2/FR-3/FR-5 → Task 1 (+ Task 2 for FR-3 wrapping); FR-4 → Task 2.
- All four plan files appear across Task 1 (interp doctest.go + test) and Task 2
  (corpus runner + test). Task 3 is verification-only (no files), matching the plan.
- No scope creep: no task touches a file outside the plan inventory.

## Ordering
- Valid DAG: Task 1 (leaf) → Task 2 (consumes RunDoctests) → Task 3 (gates).
- Codebase compiles after Task 1 (new self-contained file + test) and after Task 2
  (adds the corpus runner + test). No forward references.

## Executability
- Each task has concrete instructions referencing exact AST node paths, existing
  seams (evalExpr, Value.String, RunDoctestExec wording), and a runnable verify
  command. Each touches ≤2 files.

## Sizing
- Tasks 1 and 2 are each a single-turn unit; Task 3 is a thin final gate (acceptable
  as the mandatory whole-suite verification, not a trivial file-create).

## Findings
No CRITICAL or MAJOR findings. Ready to implement.

## Assumptions
- `RunDoctests` 3-value signature `(failures, ran, err)` lets `RunInterp` flag a
  zero-example doctest case (FR-4) without a second pass.
- The mutated-expected negative test writes a temp `.goal` file rather than mutating
  a committed fixture, keeping the corpus immutable.
