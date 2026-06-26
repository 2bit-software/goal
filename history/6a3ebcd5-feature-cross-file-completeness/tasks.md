# Tasks: Cross-file & cross-package completeness in the LSP

Complexity: **Medium** (~7 source files). Source order: T001–T003 are independent
foundations (parallelizable); T004 depends on all three; T005–T008 are tests/verify.
Tasks map to `implementation-plan.md` steps and `spec.md` FRs.

- [ ] **T001 [P]** `internal/check/check.go`: add `AnalyzePackageInDirWith(srcs []string, dir string, resolve analyze.DirResolver) ([][]Diagnostic, []error, error)` composing `BuildPackage` + `EnrichForeign(resolve)` + `runPackage`; reimplement `AnalyzePackageInDir` to delegate with `resolve=nil` (signature + existing callers unchanged). Plan Step 1 / D1.
  - **Accept**: package builds; `cmd/goal/main.go` and `internal/check/foreign_test.go` compile unchanged; new func returns enrichment `[]error`.

- [ ] **T002 [P]** `internal/lsp/uri.go` (new): add `uriToPath(uri string) (path string, ok bool)` — `file:` scheme only, percent-decoded, returns `ok=false` otherwise. Plan Step 2 / FR-003b/D6.
  - **Accept**: compiles; pure function, no IO.

- [ ] **T003 [P]** `internal/lsp/server.go`: add `fileSrc{path,src string}`, `type dirReader func(dir string)([]fileSrc,error)`, server fields `files dirReader` + `resolve analyze.DirResolver` + `analysisMu sync.Mutex`; add `NewServerWithIO(out, files, resolve)`; rewire `NewServer` to call it with `osDirReader` + `analyze.DefaultResolver`; implement non-recursive `osDirReader` (uses `project.Ext`). Plan Step 3 / FR-008/D4.
  - **Accept**: `go build ./...` passes; `cmd/goal/main.go:101` (`NewServer(out)`) unchanged; no import cycle (`lsp`→`project`,`analyze` verified clean).

- [ ] **T004 [US1][US2][US3]** `internal/lsp/diagnostics.go`: extract current `compile` body into `compileSingle` (fallback); implement package-aware `compile` per technical-spec §4 — URI→path → dir → `s.files(dir)` → open-file snapshot → buffer-wins merge-by-path → analyzed-file index (append unsaved-not-on-disk, D9) → package-name conflict guard (`project.PackageClause`, FR-005a) → `check.AnalyzePackageInDirWith(srcs, dir, s.resolve)` → log `[]error` (FR-006) → per-open-file publish to own URI/version, superseded-guarded. Serialize with `analysisMu` + re-snapshot docs (audit #2). Add `openFilesInDir`, `conflictingPackageNames`. Update `didClose` to re-`schedule` remaining open siblings (audit #5b). Plan Step 4 / FR-001..FR-010.
  - **Accept**: cross-file & foreign types resolve; non-`file:`/unreadable-dir/conflict → `compileSingle`; per-file attribution correct; analysis off the message loop.
  - **Depends on**: T001, T002, T003.

- [ ] **T005 [P]** `internal/lsp/uri_test.go` (new): table test for `uriToPath` — `file:///a/b.goal`→`/a/b.goal`, `file:///a/p%20q/x.goal`→`/a/p q/x.goal`, `untitled:x`/`http://x`→`ok=false`. FR-003b.
  - **Depends on**: T002.

- [ ] **T006** `internal/lsp` integration tests (new file, e.g. `package_test.go`) using `NewServerWithIO` with a fake in-memory `dirReader` + fake `DirResolver` (points at a fixture Go pkg dir; reuse `internal/analyze/testdata/extpkg` pattern), synchronous server (`debounce<=0`), capturing published `PublishDiagnosticsParams`:
  1. cross-file resolve + genuine unsourced-field Error (FR-001)
  2. foreign Go type resolve (FR-002)
  3. unsaved A→B refresh (FR-003/FR-010)
  4. per-URI attribution (FR-004)
  5. non-`file:` URI fallback (FR-005)
  6. package-name conflict fallback + log (FR-005a)
  7. different-dir buffer excluded (FR-003a)
  8. debounce/supersede (FR-009)
  9. parity vs `check.AnalyzePackageInDir` (FR-007)
  10. didClose re-publishes open sibling (audit #5b)
  - **Depends on**: T004.

- [ ] **T007** `internal/lsp/server_test.go`: migrate `TestServerPublishesDiagnosticsOnOpen` to `NewServerWithIO` with a no-sibling fake `dirReader` + no-op resolver so it no longer reads the real `/` (audit #4). Keep `TestServerInitializeCapabilities` as-is.
  - **Depends on**: T003 (constructor), T004 (compile).

- [ ] **T008** Verify: `go build ./...`, `go vet ./...`, `go test ./internal/lsp/... ./internal/check/... ./internal/analyze/... -count=1`. Add a `DECISIONS.md` entry if repo convention warrants. Optional manual VSCode sanity check (document if skipped).
  - **Depends on**: T001–T007.

## Traceability (FR → task)
FR-001→T004,T006 · FR-002→T001,T004,T006 · FR-003/003a/003b→T002,T004,T005,T006 ·
FR-004→T004,T006 · FR-005/005a→T004,T006 · FR-006→T001,T004 · FR-007→T006 ·
FR-008→T001,T003 · FR-009→T004,T006 · FR-010→T004,T006. No orphan tasks.
