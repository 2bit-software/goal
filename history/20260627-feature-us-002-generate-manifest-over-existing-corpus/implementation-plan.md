# Implementation Plan — US-002

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/corpus/generate.go` | `Generate(root string) (Manifest, error)` walks the three corpus locations and builds a deterministic `Manifest`. |
| `internal/corpus/generate_test.go` | Asserts the generated manifest has 51 transpile pairs and 50 check cases, and that regeneration is deterministic. |
| `cmd/corpus-gen/main.go` | Thin command that calls `Generate` and writes `corpus/manifest.json`. Target of a `go:generate` directive. |
| `corpus/manifest.json` | The generated manifest committed into the tree. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/corpus/corpus.go` | Add a `//go:generate go run ../../cmd/corpus-gen` directive (doc only); no behavior change. |

## Package Structure

```
internal/corpus/
  corpus.go          (existing: Case, Manifest, Load)
  generate.go        (new: Generate)
  generate_test.go   (new)
  testdata/manifest.json (existing fixture, untouched)
cmd/corpus-gen/
  main.go            (new)
corpus/
  manifest.json      (new, generated)
```

## Dependency Graph

1. `internal/corpus/corpus.go` types (exist).
2. `internal/corpus/generate.go` — depends on (1).
3. `cmd/corpus-gen/main.go` — depends on (2).
4. `corpus/manifest.json` — produced by running (3).
5. `internal/corpus/generate_test.go` — depends on (2).

## Interface Contracts

```go
// Generate walks the corpus rooted at root and returns a Manifest indexing
// every transpile pair and checker case. It does not move or modify any file.
func Generate(root string) (Manifest, error)
```

Walk rules:
- Transpile: `filepath.Glob(root+"/features/*/examples/*.goal")` and
  `root/testdata/*.goal` (non-recursive); include a case only when the sibling
  `<base>.go.expected` exists. `Kind=transpile, Mode=file, Normalize=gofmt`,
  `Input`/`Expected` = repo-relative slash paths.
- Check: walk `root/testdata/check` recursively for `*.goal`.
  `Kind=check, Mode=file, Normalize=none`, `Expected=""`.
- IDs: derived from the relative path (slashes/dots → hyphens), unique.
- Cases sorted by `Input` for determinism.

## Integration Points

- Reuses `corpus.Case`, `corpus.Manifest`, and the `Kind`/`Mode`/`Normalize`
  constants from `corpus.go`.
- `corpus-gen` marshals with `encoding/json` (indented) and writes
  `corpus/manifest.json`; round-trips through `corpus.Load`.

## Testing Strategy

- `generate_test.go` computes repo root as `../..` from the package dir, calls
  `Generate`, counts cases by `Kind`, asserts 51 transpile and 50 check.
- A second assertion calls `Generate` twice and compares the JSON-marshaled
  output for byte-equality (determinism).
- Stdlib `testing` only — no testify (project constraint).
