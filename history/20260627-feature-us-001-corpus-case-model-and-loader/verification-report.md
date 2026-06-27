# Verification Report — US-001

## Project gates (prd.json verifyCommands)
- `go build ./...` → PASS
- `go vet ./...` → PASS
- `go test ./... -count=1` → PASS (all packages, including new goal/internal/corpus)

## Acceptance criteria
- [x] `internal/corpus` defines `Case{ID, Kind, Input, Expected, Mode, Normalize}`
      with Kind ∈ {transpile, check, doctest}, Mode ∈ {file, package}, plus
      `Load(path) (Manifest, error)`. → corpus.go
- [x] Unit test loads a fixture manifest from `internal/corpus/testdata` and
      asserts every field of one case of each Kind. → TestLoadFixture (PASS)
- [x] Error paths covered: missing file and malformed JSON return errors, not
      panics. → TestLoadMissingFile, TestLoadMalformed (PASS)

`go test ./internal/corpus/ -v` → 3 tests PASS.

## Findings
None. No CRITICAL/MAJOR. Story complete and committed (027836c).
