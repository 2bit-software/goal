# Implementation Plan — US-005 Doctest Sidecar Runner

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/corpus/doctest_runner.go` | `RunDoctest(root, Case, Transpiler) error` — transpiles a doctest case and compares the emitted `Output.Test` sidecar against the golden, gofmt-normalizing both sides. |
| `internal/corpus/doctest_runner_test.go` | `TestDoctestRunner` — runs every `KindDoctest` case from the committed manifest against `pipeline.Transpile`; fails loudly on zero cases. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/corpus/generate.go` | When a transpile pair's golden is a doctest sidecar (golden imports `"testing"` and contains `func Test`), additionally emit a `KindDoctest` case (same Input/Expected, Mode=file, Normalize=gofmt). Add an `isDoctestSidecar(path)` helper. Transpile classification is unchanged. |
| `internal/corpus/generate_test.go` | Keep the `transpile == 51` and `check == 50` assertions (US-002). Replace the `other != 0` guard with a `doctest == 4` assertion (the 4 feature-11 sidecars). |
| `corpus/manifest.json` | Regenerated via `go run ./cmd/corpus-gen -root .` to include the 4 doctest cases. |

## Package Structure

```
internal/corpus/
  corpus.go               (unchanged — Case/Kind/Manifest/Load)
  generate.go             (MODIFIED — emit doctest cases)
  generate_test.go        (MODIFIED — doctest count assertion)
  runner.go               (unchanged — gofmtNormalize reused)
  doctest_runner.go       (NEW)
  doctest_runner_test.go  (NEW)
corpus/manifest.json      (REGENERATED)
```

## Dependency Graph

1. `generate.go` change (emit doctest cases) — foundation; nothing depends on it at compile time but the manifest regen does.
2. Regenerate `corpus/manifest.json` — depends on 1.
3. `doctest_runner.go` — depends only on existing types + `gofmtNormalize`.
4. `doctest_runner_test.go` — depends on 2 (manifest has doctest cases) and 3.
5. `generate_test.go` change — depends on 1.

## Interface Contracts

```go
// doctest_runner.go
func RunDoctest(root string, c Case, tp Transpiler) error
```

Behavior: rejects non-`KindDoctest` cases; reads `c.Input` under root; calls
`tp.Transpile`; gofmt-normalizes `out.Test` and the golden at `c.Expected`;
returns nil on equal, a case-identified error otherwise.

```go
// generate.go helper
func isDoctestSidecar(expectedPath string) bool // golden imports "testing" AND contains "func Test"
```

Reuses existing `Transpiler` / `TranspilerFunc` (runner.go) and `gofmtNormalize`
(runner.go) — no new interface needed.

## Integration Points

- `doctest_runner.go` lives in package `corpus`, reusing `gofmtNormalize` from
  `runner.go` and the `Transpiler` seam.
- `doctest_runner_test.go` reuses `manifestPath` (runner_test.go) and `repoRoot`
  (generate_test.go) constants already declared in the package.
- `TestDoctestRunner` drives `pipeline.Transpile` via `TranspilerFunc`, same as
  `TestTranspileRunner`.

## Testing Strategy

- `TestDoctestRunner`: iterate manifest, filter `KindDoctest`, run each as a
  subtest, `t.Fatalf` if zero ran.
- `generate_test.go`: assert `doctest == 4` alongside the kept `transpile == 51`
  / `check == 50`.
- Full gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
