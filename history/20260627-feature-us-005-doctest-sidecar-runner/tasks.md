# Task Breakdown — US-005 Doctest Sidecar Runner

Status: Task 1 completed, Task 2 completed, Task 3 completed. All gates green.

## Task 1: Emit doctest cases in the generator
**Files**: `internal/corpus/generate.go`, `internal/corpus/generate_test.go`,
`corpus/manifest.json`
**Spec coverage**: FR-1; AC "doctest cases exist", "counts remain true".
**Steps**:
- Add `isDoctestSidecar(expectedPath string) bool` — true when the golden
  imports `"testing"` and contains `func Test`.
- In the transpile-glob loop, when `isDoctestSidecar(expected)`, append an
  additional `KindDoctest` case (same Input/Expected, Mode=file,
  Normalize=gofmt) alongside the existing transpile case.
- Regenerate `corpus/manifest.json` (`go run ./cmd/corpus-gen -root .`).
- Update `generate_test.go`: keep `transpile == 51` and `check == 50`; replace
  the `other != 0` guard with `doctest == 4`.
**Verify**: `go test ./internal/corpus -run TestGenerate -count=1` green;
`grep -c '"doctest"' corpus/manifest.json` returns 4.

## Task 2: Implement the doctest runner
**Files**: `internal/corpus/doctest_runner.go`
**Spec coverage**: FR-2; AC "runner compares the sidecar".
**Steps**:
- Add `RunDoctest(root string, c Case, tp Transpiler) error`: reject
  non-`KindDoctest`; read input; transpile; gofmt-normalize `out.Test` and the
  golden; compare; return nil or a case-identified error.
**Verify**: `go build ./internal/corpus`.

## Task 3: Whole-corpus doctest test
**Files**: `internal/corpus/doctest_runner_test.go`
**Spec coverage**: FR-3; AC "every doctest case passes", "fail loudly on zero".
**Steps**:
- Add `TestDoctestRunner`: load manifest, iterate `KindDoctest` cases as
  subtests via `TranspilerFunc(pipeline.Transpile)`, `t.Fatalf` if zero ran.
**Verify**: `go test ./internal/corpus -run TestDoctestRunner -count=1` green.

## Final gate (all tasks)
`go build ./...`, `go vet ./...`, `go test ./... -count=1` all green.
