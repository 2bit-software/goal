# Implementation Tasks — US-006

## Task 1: Rewire the checker harness onto the corpus runner
**Status**: completed
**Files**: `internal/check/check_test.go`
**Depends on**: (none)
**Spec coverage**: FR-2, FR-3, FR-4; AC grep-check, AC build/vet/test
**Verify**: `go test ./internal/check/... -count=1` && grep finds no `testdata/check` walk or `features` path in the file

### Instructions
- Convert to external `package check_test`.
- Imports: `testing`, `goal/internal/check`, `goal/internal/corpus`.
- Add unexported consts `manifestPath = "../../corpus/manifest.json"` and `repoRoot = "../.."`.
- `TestCorpusCheck`: `corpus.Load(manifestPath)`; for each case where
  `c.Kind == corpus.KindCheck`, `t.Run(c.ID, ...)` calling
  `corpus.RunCheck(repoRoot, c, corpus.CheckerFunc(check.Analyze))` and `t.Error` on err.
  Count ran; `t.Fatalf` if zero.
- Preserve `TestRegistryRuns` verbatim except calling `check.Analyze` (it already does).
- Delete `wantRe`, `parseWants`, `runCase`, and the `testdata/check` WalkDir.

## Task 2: Rewire the pipeline harness onto the corpus runner
**Status**: completed
**Files**: `internal/pipeline/pipeline_test.go`
**Depends on**: (none)
**Spec coverage**: FR-1, FR-3, FR-4; AC grep-pipeline, AC build/vet/test
**Verify**: `go test ./internal/pipeline/... -count=1` && grep finds no `features` path or feature-dir list in the file

### Instructions
- Convert to external `package pipeline_test`.
- Imports: `testing`, `goal/internal/pipeline`, `goal/internal/corpus`.
- Add unexported consts `manifestPath = "../../corpus/manifest.json"` and `repoRoot = "../.."`.
- `TestCorpusTranspile`: for each `corpus.KindTranspile` case, `t.Run(c.ID, ...)` →
  `corpus.RunTranspile(repoRoot, c, corpus.TranspilerFunc(pipeline.Transpile))`; loud zero guard.
- `TestCorpusDoctest`: for each `corpus.KindDoctest` case, `t.Run(c.ID, ...)` →
  `corpus.RunDoctest(repoRoot, c, corpus.TranspilerFunc(pipeline.Transpile))`; loud zero guard.
- Delete the testdata/feature/feature-11 globs and the `mustFormat` helper.

## Task 3: Full-suite verification
**Status**: completed
**Files**: (none — verification only)
**Depends on**: Task 1, Task 2
**Spec coverage**: AC build/vet/test
**Verify**: `go build ./...` && `go vet ./...` && `go test ./... -count=1`; grep both files for `features` and `filepath.Join("..","..","features"` returns nothing.
