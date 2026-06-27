# Tasks — US-001 corpus Case model and loader

Status: Task 1 completed, Task 2 completed, Task 3 completed.

## Task 1: Define the corpus model and loader
- **Files**: `internal/corpus/corpus.go` (new)
- **Work**: Define `Kind`/`Mode`/`Normalize` named string types + constants,
  the `Case` struct (ID, Kind, Input, Expected, Mode, Normalize with json tags),
  the `Manifest` struct (`Cases []Case`), and `Load(path string) (Manifest, error)`
  reading + json-unmarshalling the file, wrapping errors with context.
- **Deps**: none (stdlib only).
- **Verify**: `go build ./internal/corpus/`.
- **Covers**: FR-1..FR-5.

## Task 2: Add fixture manifest
- **Files**: `internal/corpus/testdata/manifest.json` (new)
- **Work**: One transpile, one check, one doctest case, all fields populated.
- **Deps**: Task 1 (field names).
- **Verify**: valid JSON; `go vet ./internal/corpus/`.
- **Covers**: acceptance criterion "fixture with one case of each Kind".

## Task 3: Add unit tests
- **Files**: `internal/corpus/corpus_test.go` (new)
- **Work**: `TestLoadFixture` (assert every field of one case of each Kind),
  `TestLoadMissingFile`, `TestLoadMalformed`. Stdlib `testing` only, no testify.
- **Deps**: Tasks 1 and 2.
- **Verify**: `go test ./internal/corpus/ -count=1`.
- **Covers**: all acceptance criteria.
