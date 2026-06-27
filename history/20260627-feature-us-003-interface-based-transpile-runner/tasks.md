# Tasks — US-003 Interface-based transpile runner

## T1: Add Transpiler interface and RunTranspile (foundation)
- Create `internal/corpus/runner.go`.
- Define `Transpiler` interface, `TranspilerFunc` adapter, and
  `RunTranspile(root string, c Case, tp Transpiler) error`.
- Comparison: gofmt-normalize both produced output and golden; pass when golden
  equals `Output.Go` or (when non-empty) `Output.Test`.
- Depends on: nothing (uses existing pipeline + corpus model).

## T2: Add whole-corpus transpile test
- Create `internal/corpus/runner_test.go` with `TestTranspileRunner`.
- Load `../../corpus/manifest.json`, run every `KindTranspile` case against
  `TranspilerFunc(pipeline.Transpile)`; fail loudly if zero cases.
- Depends on: T1.

## T3: Verify
- `go build ./...`, `go vet ./...`, `go test ./... -count=1` all green.
- Depends on: T1, T2.
