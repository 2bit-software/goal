# Implementation Tasks

## Task 1: Add RunDoctestExec behavioral runner
**Status**: completed
**Files**: `internal/corpus/doctest_behavior_runner.go`
**Depends on**: (none)
**Spec coverage**: FR-1, FR-3
**Verify**: `go build ./...` && `go vet ./...`

Transpile a KindDoctest case, require a non-empty sidecar, write
go.mod + case.go (Output.Go) + case_test.go (Output.Test) into an isolated temp
module, run `go test ./...`, return a case-identified error with combined output
on failure.

## Task 2: Add whole-corpus behavioral doctest test
**Status**: completed
**Files**: `internal/corpus/doctest_behavior_runner_test.go`
**Depends on**: Task 1
**Spec coverage**: FR-2, FR-4
**Verify**: `go test ./internal/corpus/ -run TestDoctestExecRunner -count=1`

`-short`-skipped test iterating all KindDoctest manifest cases through
pipeline.Transpile, asserting each passes; loud `t.Fatalf` when zero ran.

## Task 3: Full verify gate
**Status**: completed
**Files**: (none)
**Depends on**: Task 2
**Spec coverage**: final acceptance criterion
**Verify**: `go build ./...` && `go vet ./...` && `go test ./... -count=1`
