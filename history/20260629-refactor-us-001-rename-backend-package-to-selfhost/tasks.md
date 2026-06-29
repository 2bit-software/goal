# Implementation Tasks — US-001

## Task 1: Copy backend sources into selfhost/backend
**Status**: completed
**Files**: selfhost/backend/{arity,backend,doctest,emit,lower,package}.goal (new)
**Depends on**: none
**Spec coverage**: FR-1, AC-1
**Verify**: `ls selfhost/backend/*.goal` shows 6 files; diff against internal/backend/*.go is empty (verbatim).

## Task 2: Split the self-contained backend tests
**Status**: completed
**Files**: internal/backend/backend_selfhost_test.go (new), internal/backend/backend_test.go (modified)
**Depends on**: none
**Spec coverage**: FR-3, AC-3
**Verify**: `task check` green (no duplicate symbols, no unused imports).

## Task 3: Add TestPortedBackendPackage gate
**Status**: completed
**Files**: internal/selfhost/port_test.go (modified)
**Depends on**: Task 1, Task 2
**Spec coverage**: FR-2, FR-3, AC-2, AC-3
**Verify**: `go test ./internal/selfhost -run TestPortedBackendPackage`.

## Task 4: Full verification
**Status**: completed
**Files**: none
**Depends on**: Task 1-3
**Spec coverage**: AC-4
**Verify**: `task check`, `task build`, `task fixpoint`.
