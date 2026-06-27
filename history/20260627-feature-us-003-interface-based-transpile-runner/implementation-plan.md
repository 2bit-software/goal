# Implementation Plan — US-003 Interface-based transpile runner

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/corpus/runner.go` | `Transpiler` interface + `TranspilerFunc` adapter + `RunTranspile(root, Case, Transpiler) error` runner that gofmt-normalizes both sides and compares (Go-or-sidecar). |
| `internal/corpus/runner_test.go` | Loads `corpus/manifest.json`, runs every `KindTranspile` case against `pipeline.Transpile` through the interface; fails if zero cases. |

### Modified Files
None. (Pipeline and the corpus model are reused unchanged.)

## Package Structure

```
internal/corpus/
  corpus.go          (existing — model + Load)
  generate.go        (existing — manifest generation)
  runner.go          (NEW — Transpiler interface + RunTranspile)
  runner_test.go     (NEW — whole-corpus transpile test)
  ...existing tests...
```

## Dependency Graph

1. `pipeline.Output`, `pipeline.Transpile` (existing, no change).
2. `corpus.Case`, `corpus.Load` (existing, no change).
3. `runner.go` — depends on 1 and 2.
4. `runner_test.go` — depends on 3 plus the committed manifest.

No import cycle: `internal/pipeline` does not import `internal/corpus`.

## Interface Contracts

```go
// Transpiler lowers goal source to its transpile output.
type Transpiler interface {
    Transpile(src string) (pipeline.Output, error)
}

// TranspilerFunc adapts a plain function to Transpiler.
type TranspilerFunc func(src string) (pipeline.Output, error)
func (f TranspilerFunc) Transpile(src string) (pipeline.Output, error)

// RunTranspile executes one KindTranspile Case against tp, reading Input and
// Expected relative to root, gofmt-normalizing both produced output and golden,
// and returns a descriptive error on any read/transpile/format failure or
// mismatch. A case passes when the normalized golden equals the normalized
// Output.Go OR (when non-empty) the normalized Output.Test.
func RunTranspile(root string, c Case, tp Transpiler) error
```

## Integration Points

- `runner.go` imports `goal/internal/pipeline` for `Output` and is supplied
  `pipeline.Transpile` via `TranspilerFunc` from the test.
- Paths: `filepath.Join(root, filepath.FromSlash(c.Input))`; root is `../..`
  from `internal/corpus`. Manifest at `../../corpus/manifest.json`.
- gofmt via `go/format`.Source.

## Testing Strategy

- `runner_test.go` `TestTranspileRunner`: load manifest, iterate `KindTranspile`
  cases as subtests, call `RunTranspile(repoRoot, c, TranspilerFunc(pipeline.Transpile))`,
  `t.Error` on failure; `t.Fatal` if zero transpile cases ran.
- Reuse the existing `repoRoot` const (`../..`) from generate_test.go (same
  package), so no duplicate declaration — define `manifestPath` only.
- Verify gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
