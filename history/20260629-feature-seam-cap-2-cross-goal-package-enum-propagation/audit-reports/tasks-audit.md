# Tasks Audit

## Coverage
- FR-1 -> T1; FR-2 -> T2+T3; FR-3 -> T2+T3; FR-4 -> T4. Gates -> T5. All FRs covered.
- Every plan file (foreign.go/.goal, lower.go/.goal, emit.go/.goal, fixtures, test)
  appears in a task. No out-of-plan files referenced.

## Ordering
- DAG: T1,T2 independent -> T3 -> T4 -> T5. Valid topological order.
- Tree compiles after T1 (additive helper + branch), after T2 (additive lowering); T3
  adds tests; T4 mirrors; T5 verifies. No task depends on a later task.

## Executability
- Each task names concrete files, helpers, and a runnable verify command.
- Each task touches <= 5 files.

## Sizing
- No trivial/oversized tasks. T3 (fixture+test) and T4 (mirror) are the largest, both
  within a single agent turn.

## Findings
No CRITICAL/MAJOR. MINOR: T4 fixpoint is the riskiest gate (build/transpile pipeline) —
flagged for careful run, already called out in the prd notes.

## Assumptions
- Mirror edits in selfhost are line-for-line analogues of the internal edits (selfhost
  files track internal at matching line numbers).
- Fixture placed under internal/backend/testdata so module-relative imports resolve.
