# Implementation Tasks

## Task 1: Document the four sema↔legacy divergences in DECISIONS.md
**Status**: completed
**Files**: `DECISIONS.md`
**Depends on**: (none)
**Spec coverage**: FR-4
**Verify**: `grep -n "US-003" DECISIONS.md`

## Task 2: Add the differential parity gate test + allowlist
**Status**: completed
**Files**: `internal/corpus/parity_test.go`
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-2, FR-3
**Verify**: `go test ./internal/corpus/ -run TestSemaLegacyParity -v`

## Task 3: Full project gates
**Status**: completed
**Files**: (none)
**Depends on**: Task 2
**Spec coverage**: all
**Verify**: `task check && task build`
