# Tasks Audit — US-006

## Coverage
- FR-1 → Task 2; FR-2 → Task 1; FR-3 → Tasks 1+2; FR-4 → Tasks 1+2;
  AC build/vet/test → Task 3. All spec items covered.
- Both plan files (`check_test.go`, `pipeline_test.go`) appear in tasks. No
  out-of-plan files referenced (no scope creep).

## Ordering
- Tasks 1 and 2 are independent (different files, different packages); each
  leaves the codebase compiling. Task 3 depends on both. Valid DAG, no cycles.

## Executability
- Each task names exact file, package decl, imports, symbols to add/remove, and a
  concrete verify command. Each modifies a single file (Task 3 is verify-only).

## Sizing
- Each task is a single-file edit completable in one turn — appropriately sized.

## Findings
No CRITICAL, MAJOR, or MINOR findings.

## Assumptions
- Tasks 1 and 2 may be executed in either order or together in one turn since they
  are independent; the recommended approach is to do both then run Task 3.
- The loud zero-case guard uses `t.Fatalf`, matching the existing corpus runner tests.
