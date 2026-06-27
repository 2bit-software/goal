# US-001 — Corpus Case model and loader

## Functional requirements
- A new package `internal/corpus` provides a runner-independent model of golden
  test cases so the test suite is decoupled from package/file layout.
- `Case` describes one golden case with fields:
  - `ID` — stable identifier for the case.
  - `Kind` — one of `transpile`, `check`, `doctest`.
  - `Input` — path to the `.goal` input source (relative to manifest root).
  - `Expected` — path to the expected output / sidecar (may be empty for cases
    whose expectation is inline, e.g. `// want` markers).
  - `Mode` — one of `file`, `package`.
  - `Normalize` — the comparison normalization to apply (e.g. `gofmt`, `none`).
- A `Manifest` aggregates the loaded cases (and the root they resolve against).
- `Load(path) (Manifest, error)` reads a manifest file (JSON) from `path` and
  returns the populated `Manifest`.

## Acceptance criteria
- `internal/corpus` defines `Case{ID, Kind(transpile|check|doctest), Input,
  Expected, Mode(file|package), Normalize}` and `Load(path)(Manifest,error)`.
- A unit test loads a small fixture manifest from
  `internal/corpus/testdata` and asserts every field of one case of each Kind.
- `go build ./...`, `go vet ./...`, `go test ./...` all pass.

## Out of scope
- Generating a manifest over the real corpus (US-002).
- Any runner / execution of cases (US-003+).
- Moving or rewriting existing golden files.

## Open questions
- None. Kind and Mode are modeled as small string-backed types with validation
  to keep the manifest human-authorable and JSON-friendly.
