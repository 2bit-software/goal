# Implementation Plan — US-008 Run Doctests Behaviorally

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/corpus/doctest_behavior_runner.go` | `RunDoctestExec(root, Case, Transpiler)` — transpile a KindDoctest case, write package + sidecar into a temp module, run `go test ./...`. |
| `internal/corpus/doctest_behavior_runner_test.go` | `TestDoctestExecRunner` — drive all KindDoctest cases through pipeline.Transpile; `-short`-skipped; loud zero-case guard. |

### Modified Files
None. (Reuses existing `Transpiler`/`TranspilerFunc`, `Case`, `Load`,
`manifestPath`, `repoRoot`.)

## Design

`RunDoctestExec`:
1. Guard `c.Kind == KindDoctest` (mirror RunDoctest's wrong-kind error).
2. Read `c.Input` relative to `root`; transpile via `tp`.
3. Require non-empty `out.Test` (the doctest sidecar); error loudly otherwise.
4. `os.MkdirTemp` a module dir; `defer os.RemoveAll`.
5. Write `go.mod` (`module goalcorpus` / `go 1.26` — same as RunCompile),
   `case.go` (`out.Go`), `case_test.go` (`out.Test`).
6. `exec.Command("go", "test", "./...")`, `cmd.Dir = tmp`, `CombinedOutput()`;
   on error return a case-identified error including combined output + both
   generated sources.

`TestDoctestExecRunner`: skip under `-short`; `Load(manifestPath)`; iterate
`m.Cases` filtering `KindDoctest`; `t.Run(c.ID, ...)` calling
`RunDoctestExec(repoRoot, c, tp)`; `t.Fatalf` if zero ran.

## Test Strategy

- Whole-corpus behavioral test asserts all 4 feature-11 doctest cases pass
  `go test`.
- Verify gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.

## Reuse

- Temp-module recipe and go.mod constant style from `behavior_runner.go`.
- Case selection / interface seam from `doctest_runner.go` + `runner.go`.
- Test skeleton from `behavior_runner_test.go`.
