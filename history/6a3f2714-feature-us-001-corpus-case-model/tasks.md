# Tasks — US-001 corpus Case model and loader

Complexity: Simple (3 files).

- [ ] T001 [US-001] Create `internal/corpus/corpus.go` with `Kind`/`Mode`/`Normalize`
  string types + consts, `Case`, `Manifest`, and `Load(path)(Manifest,error)`
  (JSON read, root resolution, Kind/Mode validation, defaults).
- [ ] T002 [US-001] Create `internal/corpus/testdata/manifest.json` fixture with one
  transpile, one check, and one doctest case.
- [ ] T003 [US-001] Create `internal/corpus/corpus_test.go` asserting every field of
  one case per Kind from the loaded fixture.
- [ ] T004 [US-001] Run `go build ./...`, `go vet ./...`, `go test ./...` — all green.
