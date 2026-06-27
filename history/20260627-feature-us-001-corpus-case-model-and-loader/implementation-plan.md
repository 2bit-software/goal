# Implementation Plan — US-001 corpus Case model and loader

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/corpus/corpus.go` | Defines `Kind`, `Mode`, `Normalize` named types + constants, the `Case` struct, the `Manifest` struct, and `Load(path string) (Manifest, error)`. |
| `internal/corpus/corpus_test.go` | Loads the fixture manifest and asserts every field of one case of each Kind, plus the error path for a missing/malformed file. |
| `internal/corpus/testdata/manifest.json` | Small fixture manifest with one transpile, one check, and one doctest case. |

### Modified Files
None. This is a new, self-contained package.

## Package Structure

```
internal/
  corpus/
    corpus.go
    corpus_test.go
    testdata/
      manifest.json
```

## Dependency Graph

1. `internal/corpus/corpus.go` — model + loader (depends only on stdlib `encoding/json`, `os`, `fmt`).
2. `internal/corpus/testdata/manifest.json` — fixture (depends on field names from 1).
3. `internal/corpus/corpus_test.go` — depends on 1 and 2.

## Interface Contracts

```go
package corpus

type Kind string
const (
    KindTranspile Kind = "transpile"
    KindCheck     Kind = "check"
    KindDoctest   Kind = "doctest"
)

type Mode string
const (
    ModeFile    Mode = "file"
    ModePackage Mode = "package"
)

type Normalize string
const (
    NormalizeNone  Normalize = "none"
    NormalizeGofmt Normalize = "gofmt"
)

type Case struct {
    ID        string    `json:"id"`
    Kind      Kind      `json:"kind"`
    Input     string    `json:"input"`
    Expected  string    `json:"expected"`
    Mode      Mode      `json:"mode"`
    Normalize Normalize `json:"normalize"`
}

type Manifest struct {
    Cases []Case `json:"cases"`
}

func Load(path string) (Manifest, error)
```

## Integration Points

None in this story. Later stories (US-002 generates the manifest; US-003/004/005
build runners that consume `Manifest`/`Case`). The package is standalone now.

## Testing Strategy

- `internal/corpus/corpus_test.go` (stdlib `testing` only — no testify):
  - `TestLoadFixture`: load `testdata/manifest.json`; assert it has 3 cases; pick
    the transpile, check, and doctest cases by ID and assert every field
    (ID, Kind, Input, Expected, Mode, Normalize) equals the fixture content.
  - `TestLoadMissingFile`: `Load("testdata/does-not-exist.json")` returns an error.
  - `TestLoadMalformed`: a temp file with invalid JSON returns an error (no panic).
