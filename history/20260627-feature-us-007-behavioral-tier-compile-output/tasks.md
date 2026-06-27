# Implementation Tasks — US-007

## Task 1: Add the behavioral compile runner
**Status**: completed
**Files**: `internal/corpus/behavior_runner.go`
**Depends on**: (none)
**Spec coverage**: FR-1, AC-1, AC-3
**Verify**: `go build ./... && go vet ./...`

### Instructions
- New file `internal/corpus/behavior_runner.go`, `package corpus`.
- Implement `func RunCompile(root string, c Case, tp Transpiler) error`:
  - Reject non-`KindTranspile` cases with a descriptive error (mirror
    `RunTranspile`'s guard).
  - Read `c.Input` relative to `root` (`filepath.Join(root, filepath.FromSlash(c.Input))`).
  - `tp.Transpile(string(src))` → `out`; wrap errors as `corpus: case %q: ...`.
  - `os.MkdirTemp("", "goal-corpus-compile-*")`; `defer os.RemoveAll`.
  - Write `go.mod` (`module goalcorpus\n\ngo 1.26\n`) and `out.Go` as a `.go`
    file in the temp dir.
  - Run `go build ./...` then `go vet ./...` via `exec.Command("go", ...)` with
    `cmd.Dir = tmp`, capturing combined output; on failure return a
    case-identified error including the tool output.
  - Return nil on success. Never write into the source tree.

## Task 2: Whole-corpus behavioral test
**Status**: completed
**Files**: `internal/corpus/behavior_runner_test.go`
**Depends on**: Task 1
**Spec coverage**: FR-2, AC-2, AC-4
**Verify**: `go test ./internal/corpus/ -run TestCompileRunner -count=1`

### Instructions
- New file `internal/corpus/behavior_runner_test.go`, `package corpus`.
- `TestCompileRunner`: skip under `testing.Short()`.
- Load `../../corpus/manifest.json` via `Load` (reuse existing repoRoot `../..`).
- Range cases; for each `KindTranspile` case run a subtest
  (`t.Run(c.ID, ...)`) calling `RunCompile(repoRoot, c, TranspilerFunc(pipeline.Transpile))`;
  `t.Errorf` on error.
- `t.Fatalf` if zero transpile cases were seen (loud zero-case guard).
