# Progress Log

All tasks complete. `go build ./...`, `go vet ./internal/lsp/...`, `gofmt -l`, and the full
`go test ./...` suite pass.

### T001 — check.AnalyzePackageInDirWith — Complete
- Files: `internal/check/check.go`
- Added resolver-injectable `AnalyzePackageInDirWith(srcs, dir, resolve) ([][]Diagnostic, []error, error)`; `AnalyzePackageInDir` now delegates with `resolve=nil`. Existing callers unchanged.

### T002 — uriToPath — Complete
- Files: `internal/lsp/uri.go` (new)
- `file:`-only, percent-decoding, `filepath.Clean`; non-file/empty → `ok=false`.

### T003 — IO seams + constructors — Complete
- Files: `internal/lsp/server.go`
- Added `fileSrc`, `dirReader`, server fields `files`/`resolve`/`analysisMu`, `NewServerWithIO`, prod `osDirReader` (non-recursive). `NewServer` delegates with real IO. `didClose` now re-schedules open siblings.

### T004 — package-aware compile — Complete
- Files: `internal/lsp/diagnostics.go`
- `compile` resolves the file's package dir, overlays open buffers on disk (buffer wins, plus unsaved-not-on-disk), guards on package-name conflict, runs `AnalyzePackageInDirWith`, logs foreign errors (non-fatal), and publishes per-open-file to each URI. Serialized by `analysisMu` with a docs re-snapshot. `compileSingle` is the fallback.

### T005 — uriToPath test — Complete
- Files: `internal/lsp/uri_test.go` (new)

### T006 — LSP integration + parity tests — Complete
- Files: `internal/lsp/package_test.go` (new)
- 10 tests: cross-file resolve, cross-file unsourced-field error, foreign derive resolve (via injected resolver → `check/testdata/extpkg`), edit-refreshes-open-sibling, per-file attribution, non-file fallback, package-conflict fallback, open-files-in-dir dir exclusion, close-refreshes-sibling, CLI parity.

### T007 — migrate existing test — Complete
- Files: `internal/lsp/server_test.go`
- `TestServerPublishesDiagnosticsOnOpen` now uses `NewServerWithIO` with an empty fake dir reader, no longer reading the real `/`.

### T008 — verify — Complete
- `go test ./...` all green; gofmt clean; vet clean.

## Decisions / Notes
- Foreign tests inject a fake `DirResolver` pointing at the existing `extpkg` fixture, so no
  go toolchain / network is needed; parsing the fixture's `.go` is pure `go/parser`.
- D5 caching not implemented (rely on debounce); revisit only if a real package is laggy.
- Symlinked package dirs remain a known limitation (paths are `Clean`ed, not `EvalSymlinks`d).
