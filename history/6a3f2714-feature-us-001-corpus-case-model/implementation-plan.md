# Implementation plan — US-001 corpus Case model and loader

## Files
- `internal/corpus/corpus.go` — new package with `Kind`, `Mode`, `Normalize`,
  `Case`, `Manifest`, and `Load`.
- `internal/corpus/corpus_test.go` — loads the fixture manifest and asserts every
  field of one case of each Kind.
- `internal/corpus/testdata/manifest.json` — small fixture manifest with one
  `transpile`, one `check`, and one `doctest` case.

## Design
- `Kind string` with consts `KindTranspile="transpile"`, `KindCheck="check"`,
  `KindDoctest="doctest"`; `Mode string` with `ModeFile="file"`,
  `ModePackage="package"`; `Normalize string` with `NormalizeGofmt="gofmt"` and
  `NormalizeNone="none"`.
- `Case` struct with JSON tags: `ID, Kind, Input, Expected, Mode, Normalize`.
- `Manifest` struct: `Root string` (dir containing the manifest, set by Load) and
  `Cases []Case`.
- `Load(path) (Manifest, error)`: read file, `json.Unmarshal` into a wire struct
  holding `Cases`, set `Root = filepath.Dir(path)`, validate each Case's Kind and
  Mode (return a descriptive error on unknown values), and return the Manifest.
- Default behavior: if a Case omits `Mode`, default to `ModeFile`; if it omits
  `Normalize`, default to `NormalizeNone`. Keeps manifests terse and forward
  compatible (US-002 generates real manifests).

## Verification
- `go build ./...`, `go vet ./...`, `go test ./...`.
- Test asserts ID/Kind/Input/Expected/Mode/Normalize for each of the three Kinds.

## Reuse note
No existing manifest/loader exists (grep for `corpus` found only doc references in
REWRITE-ARCHITECTURE.md). Stdlib `encoding/json` + `path/filepath` only — the repo
is zero-dependency and tests use stdlib `testing`.
