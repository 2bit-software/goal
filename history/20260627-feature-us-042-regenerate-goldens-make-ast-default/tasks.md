# Implementation Tasks — US-042

## Task 1: Add -update-goldens regeneration test
**Status**: completed
**Files**: internal/corpus/update_goldens_test.go (new)
**Depends on**: none
**Spec coverage**: FR-2
**Verify**: `go test ./internal/corpus -run TestUpdateGoldens` (no-op without flag)

## Task 2: Regenerate goldens from the AST backend
**Status**: completed
**Files**: features/**/*.go.expected, testdata/**/*.go.expected (content)
**Depends on**: Task 1
**Spec coverage**: FR-2
**Verify**: `go test ./internal/corpus -run TestUpdateGoldens -update-goldens` then `git status` shows only golden content changes

## Task 3: Switch exact-tier tests to the AST backend
**Status**: completed
**Files**: internal/corpus/runner_test.go, internal/corpus/doctest_runner_test.go, internal/pipeline/pipeline_test.go
**Depends on**: Task 2
**Spec coverage**: FR-3
**Verify**: `go test ./internal/corpus ./internal/pipeline -run 'Transpile|Doctest'`

## Task 4: Flip CLI default engine to AST + docs
**Files**: cmd/goal/main.go, cmd/goal/main_test.go, AI-KNOWLEDGE-BOOTSTRAP.md
**Depends on**: none
**Spec coverage**: FR-1, FR-4, FR-5
**Verify**: `go test ./cmd/goal` ; `go run ./cmd/goal ai | diff - AI-KNOWLEDGE-BOOTSTRAP.md`

## Task 5: Full verify
**Files**: (none)
**Depends on**: Task 3, Task 4
**Spec coverage**: all ACs + behavioral gate
**Verify**: `go build ./...` ; `go vet ./...` ; `go test ./... -count=1`
