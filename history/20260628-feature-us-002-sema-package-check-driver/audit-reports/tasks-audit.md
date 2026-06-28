# Tasks Audit — US-002

## Coverage
- FR-1/FR-2/FR-4 -> Task 1; FR-1/FR-2/FR-3 -> Task 2. All FRs covered.
- Plan files package.go (Task 1) and package_test.go (Task 2) both present.
- No scope creep — only the two planned files are touched.

## Ordering
- Valid DAG: Task 1 (no deps) -> Task 2 (depends on Task 1). Codebase compiles
  after Task 1 (driver added); test added in Task 2.

## Executability
- Each task has concrete instructions referencing real seams and a runnable
  verify command (`go build ./...`, `go test ./internal/sema/ -count=1`).
- Each task touches a single file (<5).

## Sizing
- Both tasks are single-turn sized and non-trivial.

No CRITICAL/MAJOR findings. Recommend pass.

## Assumptions
- The control assertion in Task 2 contrasts Warning vs Error to prove the
  enrichment dependency, rather than asserting only the enriched Error.
