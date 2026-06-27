# Tasks Audit (US-006)

## Coverage
Every FR maps to a task: FR-1/FR-3 → T3 (DeclStmt), FR-2/FR-4/FR-5 → T3
(AssignStmt) backed by T1 (Assign) and T2 (compound/zero helpers), FR-6 → T2
(Ident Lookup) + T3 (assign error). All three plan files (env.go, eval.go,
interp.go) plus a test file are covered. No scope creep.

## Ordering
T1 and T2 are independent and compile standalone; T3 depends on both; T4 on T3.
Valid DAG, no forward references, compiles after each task.

## Executability
Each task names a specific file, concrete instructions referencing existing
symbols (applyBinary, Lookup, NotFoundError), and a runnable verify step. Each
touches ≤2 files.

## Sizing
Four right-sized tasks; none trivial, none the-whole-feature. No CRITICAL,
MAJOR, or MINOR findings.

## Assumptions
- One new test file (assign_test.go) rather than appending to interp_test.go,
  to keep the assignment suite cohesive (existing tests use per-feature files).
