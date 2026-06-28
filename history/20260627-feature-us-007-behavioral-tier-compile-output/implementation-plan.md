# Implementation Plan — US-007 behavioral tier compile output

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/corpus/behavior_runner.go` | `RunCompile(root, Case, Transpiler) error`: transpile the case, write `Output.Go` + a minimal `go.mod` into an isolated temp module, run `go build` then `go vet`, returning a case-identified error with tool output on failure. |
| `internal/corpus/behavior_runner_test.go` | `package corpus` external-style test (same dir) driving every `KindTranspile` case in the manifest through `RunCompile(...,TranspilerFunc(pipeline.Transpile))`; asserts each builds + vets; `t.Fatalf` on zero cases; skips under `-short`. |

### Modified Files
| File | Changes |
|------|---------|
| (none) | Additive only — no existing source modified. |

## Package Structure

```
internal/corpus/
  runner.go              (existing) Transpiler, TranspilerFunc, RunTranspile
  behavior_runner.go     (new)      RunCompile
  behavior_runner_test.go(new)      TestCompileRunner
```

## Dependency Graph

1. `behavior_runner.go` — depends only on existing `Transpiler`/`Case`/`pipeline.Output` and stdlib (`os`, `os/exec`, `path/filepath`, `fmt`).
2. `behavior_runner_test.go` — depends on 1, the manifest, and `pipeline.Transpile`.

## Interface Contracts

```go
// RunCompile executes one KindTranspile Case behaviorally: it transpiles the
// case input via tp, writes Output.Go into a fresh temp module, and runs
// `go build` then `go vet` against it. Returns a case-identified error (with the
// go tool's combined output) on any failure; nil when the generated Go builds
// and vets cleanly.
func RunCompile(root string, c Case, tp Transpiler) error
```

Test (existing convention: repoRoot `../..`, manifest `../../corpus/manifest.json`):

```go
func TestCompileRunner(t *testing.T) {
    if testing.Short() { t.Skip("behavioral tier spawns the go toolchain") }
    // load manifest, range KindTranspile cases, RunCompile each, t.Errorf on err
}
```

## Integration Points

- Reuses the existing `Transpiler` seam from `internal/corpus/runner.go`; no new
  interface introduced.
- Mirrors `cmd/goal/main.go` temp-dir + `exec.Command("go", ...)` pattern, but
  with a standalone temp module (go.mod + one .go file) per case rather than a
  build overlay.
- Manifest loaded with the same `repoRoot`/`manifestPath` constants used by the
  sibling runner tests.

## Testing Strategy

- One whole-corpus behavioral test, subtests keyed by case ID for clear failure
  attribution.
- Skip under `-short`; the full `go test ./... -count=1` gate runs it.
- Loud zero-case guard (`t.Fatalf`) matching the other corpus runner tests.
