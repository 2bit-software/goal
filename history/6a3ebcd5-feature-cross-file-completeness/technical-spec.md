# Technical Spec: Cross-file & cross-package completeness in the LSP

Implementation design for the spec. Reuses existing analysis; adds only the editor wiring,
two IO seams, and one resolver-injectable `check` entry. Stdlib-only (no new deps).

## Change set overview

| Area | File(s) | Change |
|---|---|---|
| Resolver-injectable check entry | `internal/check/check.go` | Add `AnalyzePackageInDirWith`; make `AnalyzePackageInDir` delegate |
| URI→path helper | `internal/lsp/uri.go` (new) | `uriToPath(uri) (string, bool)` |
| IO seams + constructor | `internal/lsp/server.go` | `fileSrc`, `dirReader`, server fields, `NewServerWithIO`, prod `osDirReader` |
| Package-aware compile | `internal/lsp/diagnostics.go` | Replace single-file `compile` with package view + per-file publish + fallback |
| Tests | `internal/lsp/*_test.go`, fixtures | Integration + parity + fallback tests |

## 1. `internal/check` — resolver-injectable entry (D1)

`AnalyzePackageInDir` hardcodes `nil` resolver and discards `EnrichForeign`'s `[]error`,
so it can neither be faked in tests (FR-008) nor log foreign failures (FR-006). Add:

```go
// AnalyzePackageInDirWith is AnalyzePackageInDir with the import resolver injected and the
// per-import enrichment errors surfaced, so callers (the language server) can supply a fake
// resolver in tests and log resolution failures. A nil resolve uses analyze.DefaultResolver.
func AnalyzePackageInDirWith(srcs []string, dir string, resolve analyze.DirResolver) ([][]Diagnostic, []error, error) {
    tables := analyze.BuildPackage(srcs)
    ferrs := analyze.EnrichForeign(tables, srcs, dir, resolve)
    diags, err := runPackage(srcs, tables)
    return diags, ferrs, err
}
```

Reimplement the existing function to delegate (signature + two callers unchanged):

```go
func AnalyzePackageInDir(srcs []string, dir string) ([][]Diagnostic, error) {
    diags, _, err := AnalyzePackageInDirWith(srcs, dir, nil)
    return diags, err
}
```

## 2. `internal/lsp/uri.go` (new) — URI→path (D6, FR-003b)

Only URI→path is needed; overlay matching is done by comparing paths (no path→URI round
trip required — more robust than reconstructing the editor's URI string).

```go
// uriToPath converts a document URI to a local filesystem path. It returns ok=false for any
// URI that is not a usable file: path (other schemes, parse failure), which the caller treats
// as "no package directory" and falls back to single-file analysis.
func uriToPath(uri string) (path string, ok bool) {
    u, err := url.Parse(uri)
    if err != nil || u.Scheme != "file" {
        return "", false
    }
    p := u.Path // net/url percent-decodes; leading slash form file:///abs → /abs
    if p == "" {
        return "", false
    }
    return filepath.Clean(p), true
}
```

(Windows drive-letter handling is out of scope — darwin/linux primary. Symlinked dirs are a
known limitation: paths are `Clean`ed, not `EvalSymlinks`d — noted in spec edge cases.)

## 3. `internal/lsp/server.go` — IO seams + constructor (D4, FR-008)

```go
// fileSrc is one on-disk .goal file: its cleaned path and contents.
type fileSrc struct {
    path string
    src  string
}

// dirReader lists and reads the .goal files of a single directory. Injected so tests need no
// real filesystem; the production reader is osDirReader.
type dirReader func(dir string) ([]fileSrc, error)
```

Server gains two fields:

```go
type Server struct {
    // ...existing...
    files   dirReader
    resolve analyze.DirResolver
}
```

Constructors:

```go
// NewServer returns a server wired to the real filesystem and the default import resolver.
func NewServer(out io.Writer) *Server {
    return NewServerWithIO(out, osDirReader, analyze.DefaultResolver)
}

// NewServerWithIO is NewServer with the filesystem and import-resolution seams injected, for
// tests that drive the server without real disk or the go toolchain.
func NewServerWithIO(out io.Writer, files dirReader, resolve analyze.DirResolver) *Server {
    return &Server{ out: out, docs: ..., timers: ..., debounce: ..., files: files, resolve: resolve }
}
```

Production reader (single directory, not a recursive `project.Discover`):

```go
// osDirReader returns the .goal files directly in dir (non-recursive), each read from disk.
func osDirReader(dir string) ([]fileSrc, error) {
    entries, err := os.ReadDir(dir)
    if err != nil { return nil, err }
    var out []fileSrc
    for _, e := range entries {
        if e.IsDir() || !strings.HasSuffix(e.Name(), project.Ext) { continue }
        p := filepath.Join(dir, e.Name())
        b, err := os.ReadFile(p)
        if err != nil { return nil, err }
        out = append(out, fileSrc{path: filepath.Clean(p), src: string(b)})
    }
    return out, nil
}
```

## 4. `internal/lsp/diagnostics.go` — package-aware compile (FR-001..FR-010)

Replace the body of `compile` (or have `compileLatest` call a new `compilePackage`). Algorithm:

```
compile(uri, text, version):
  path, ok := uriToPath(uri)
  if !ok:                      # FR-005: no directory
      compileSingle(uri, text, version); return
  dir := filepath.Dir(path)
  disk, err := s.files(dir)
  if err != nil:               # FR-005: dir unreadable
      logf(...); compileSingle(uri, text, version); return

  # --- assemble package view (FR-003, FR-003a, D9) ---
  open := s.openFilesInDir(dir)   # snapshot: map[cleanPath]{uri, text, version}; uses uriToPath, skips non-file
  open[path] = {uri, text, version}   # analyzed buffer is authoritative & guaranteed present
  # order: union of disk paths + any open-only paths (e.g. unsaved analyzed file), sorted by path
  views := mergeByPath(disk, open)    # buffer text wins over disk; []srcEntry{path, src, openURI, openVersion, isOpen}
  sort views by path
  srcs := [v.src for v in views]
  analyzedIdx := index of path in views

  # --- package-name conflict guard (FR-005a, D8) ---
  if conflictingPackageNames(srcs):   # project.PackageClause across srcs, ignoring empty
      logf(...); compileSingle(uri, text, version); return

  # --- analyze (FR-001, FR-002) ---
  perFile, ferrs, err := check.AnalyzePackageInDirWith(srcs, dir, s.resolve)
  if err != nil: logf(...); return     # internal checker bug, not a rejected program
  for _, fe := range ferrs: logf("foreign resolve: %v", fe)   # FR-006 non-fatal

  # --- publish for every OPEN file in the package (FR-004, FR-010, D7) ---
  for v in views where v.isOpen:
      if s.superseded(v.openURI, v.openVersion): continue
      out := toLSP-convert(perFile[v.index], v.src)
      s.publish(v.openURI, v.openVersion, out)
```

Key points:
- **Per-file attribution (FR-004)**: `perFile[i]` aligns with `srcs[i]` (verified:
  `runPackage` preserves input order). Publish each open file's slice to its own URI.
- **Sibling refresh (FR-010/D7)**: one package run publishes refreshed diagnostics for the
  analyzed file *and every open sibling*, so fixing A clears B's stale deferrals without
  touching B. No extra scheduling needed — the fan-out is the publish loop.
- **Supersede per URI**: each open file is published at the version read when the view was
  snapshotted; if it changed mid-flight, `superseded` drops the stale publish for that URI.
- **Fallback (FR-005/005a)**: `compileSingle` is today's exact behavior
  (`check.Analyze(text)` → publish to the one URI), preserving the `file:///x.goal` test path.
- **toLSP** conversion is unchanged; it takes the file's own source for line/col mapping, so
  pass `v.src` per file.

Helper sketches:
```go
func (s *Server) openFilesInDir(dir string) map[string]openFile { /* iterate s.docs under lock, uriToPath each, keep those whose filepath.Dir==dir */ }
func conflictingPackageNames(srcs []string) bool { /* project.PackageClause; first non-empty sets name; any differing non-empty → true */ }
```

**didClose symmetry (audit #5b, D7):** when a file closes, its open siblings' diagnostics can
go stale (the closed file reverts to disk content, which may change a sibling's result). After
`didClose` deletes the doc and clears its own diagnostics, it MUST `schedule` re-analysis of
the still-open files in the same directory (reuse `openFilesInDir`, exclude the closed URI),
so the close-time fan-out mirrors the open/change-time fan-out.

## 5. Concurrency (FR-009, SC-005) — serialize package analysis

`schedule` fires each file's compile on its own `time.AfterFunc` goroutine with no
serialization. The sibling fan-out publishes B from A's run; `superseded(B, vB)` only detects
a newer B version, so it CANNOT drop a publish of B computed from a **stale view of A** (B's
own version never moved). Two concurrent runs can then race and the older one may land last,
leaving stale squiggles on B (audit MAJOR #2).

Fix: a dedicated `analysisMu sync.Mutex` (separate from `s.mu`) serializes the
package-analysis path. Acquire `analysisMu` at the top of `compile`, then **snapshot `s.docs`
after acquiring it** (briefly under `s.mu`), so the last compile to run reads the freshest
buffers and publishes last → its result is the final state; earlier runs' publishes are
overwritten by the newest. Release `s.mu` before the slow `s.files`/`go list`/analysis work
(FR-009); hold only `analysisMu` across it. (`analysisMu` serializes analysis, not message
reading — the Run loop keeps reading.) Per-URI publish-generation counters are an optional
extra guard but the serialize-and-re-snapshot approach is sufficient and simpler.

- No new caching in v1 (D5/OQ-1); rely on the 200ms debounce + `superseded` + `analysisMu`.

## 6. Fallback reality + the synchronous test path (audit MAJOR #4)

Correction to a wrong assumption: `file:///x.goal` does **not** hit `compileSingle`.
`uriToPath` returns `("/x.goal", true)`, `dir="/"`, and the **production** `osDirReader`
would `os.ReadDir("/")` on the real filesystem root. The single-file fallback triggers ONLY
for (a) a non-`file:`/unparsable URI, or (b) `s.files(dir)` returning an error, or (c) a
package-name conflict. A resolvable dir with no `.goal` siblings is NOT a fallback — it is a
package of one (the analyzed buffer), which still yields today's diagnostics.

Consequence for tests: the existing `TestServerPublishesDiagnosticsOnOpen` constructs with
`NewServer` (real `osDirReader`) and `file:///x.goal`; with the new path it would read the
real `/`. It happens to still pass (no `.goal` in `/`, source has no imports so no `go list`),
but it now does real root-FS IO — fragile. **Migrate that test to `NewServerWithIO` with a
fake `dirReader` (returning no siblings) and a no-op resolver.** New tests use the same harness.

## Test plan (stdlib `testing`, no testify)

- `internal/lsp`: `NewServerWithIO` with a fake `dirReader` (in-memory file map) and a fake
  `analyze.DirResolver` (points at a fixture Go pkg dir, e.g. reuse
  `internal/analyze/testdata/extpkg` or a local copy). Drive `didOpen`/`didChange` for two
  URIs in one dir; capture published `PublishDiagnosticsParams` from the server's `out`.
  - cross-file resolve (FR-001), foreign resolve (FR-002), unsaved A→B refresh (FR-003/FR-010),
    per-URI attribution (FR-004), non-file URI fallback (FR-005), package-conflict fallback
    (FR-005a), different-dir buffer excluded (FR-003a), debounce/supersede (FR-009).
- `internal/lsp/uri_test.go`: `uriToPath` for `file:///a/b.goal`, percent-encoded space, and
  non-file URIs (FR-003b).
- Parity (FR-007): assert the LSP package path produces the same diagnostic set as
  `check.AnalyzePackageInDir` for a fixture package.
- `internal/check`: a test that `AnalyzePackageInDirWith` with a fake resolver resolves a
  foreign type, and that `AnalyzePackageInDir` still behaves as before (delegation).
