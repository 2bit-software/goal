---
status: complete
updated: 2026-06-26
---

# Technical Research: Cross-file & cross-package completeness in the LSP

## Executive Summary

The `completeness deferred` flood in VSCode comes from the language server analyzing each
file in isolation (`internal/lsp/diagnostics.go:12` → `check.Analyze(text)` →
`analyze.Build(src)` over one buffer). The cross-file and cross-package (non-goal Go)
resolution the user wants **already exists** and ships in the `goal check` CLI via
`check.AnalyzePackageInDir` (= `analyze.BuildPackage` + `analyze.EnrichForeign` +
`runPackage`). The work is to route the editor through that same composition while overlaying
unsaved buffer text and attributing per-file diagnostics — wiring, not new analysis.

## Findings

### Codebase Context

**Root-cause call chain:**
```
VSCode ext (editors/vscode/src/extension.ts) → spawns `goal lsp` over stdio
 → cmd/goal/main.go:101  case "lsp": lsp.NewServer(out).Run(os.Stdin)
   → internal/lsp/server.go  didOpen/didChange → schedule → compileLatest
     → internal/lsp/diagnostics.go:12  check.Analyze(text)        ← SINGLE FILE
       → internal/check/check.go:150  Analyze(src){ Run(src, analyze.Build(src)) }
```
`analyze.Build` is lexical over one buffer; it cannot see sibling `.goal` files or imports —
so `internal/check/convert.go:111` (`unresolved-derive-type`), `fields.go:116`
(`unresolved-literal-type`), and the other completeness checks defer.

**Machinery already present (reuse, don't reinvent):**

| Capability | Function | Location |
|---|---|---|
| Cross-FILE table merge | `analyze.BuildPackage([]string) *Tables` + `Tables.Merge` | `internal/analyze/analyze.go:198,212` |
| Cross-PACKAGE Go enrichment | `analyze.EnrichForeign(t *Tables, srcs []string, fromDir string, resolve DirResolver) []error` | `internal/analyze/foreign.go:66` |
| Import-path → directory (injectable) | `analyze.DirResolver` (`func(importPath, fromDir string) (string, error)`) / `DefaultResolver` | `internal/analyze/foreign.go:57,441` |
| CLI composition we mirror | `check.AnalyzePackageInDir(srcs, dir) ([][]Diagnostic, error)` | `internal/check/check.go:169` |
| Per-file diagnostics | `check.runPackage(srcs, tables) ([][]Diagnostic, error)` (unexported) | `internal/check/check.go:177` |
| Package discovery | `project.Discover(root) ([]*Package, error)` — `Package{Dir,Name,Files{Path,Name,Src}}` | `internal/project/project.go:53` |
| Import parsing | `analyze.ParseImports(src) []ImportSpec` (`{Alias, Path}`) | `internal/analyze/foreign.go:146` |

**LSP server surface today (`internal/lsp/server.go`):**
- `Server.docs map[string]*doc` (URI → `{text, version}`) is the authoritative unsaved-buffer store, mutex-guarded.
- `compileLatest → compile` (`diagnostics.go`); `superseded` drops stale runs; `schedule` debounces (200ms; `<=0` ⇒ synchronous, used by `server_test.go`).
- Only constructor is `NewServer(out io.Writer)` — **no IO/filesystem seam injected**.
- Tests use `file:///x.goal`; **no URI→path helper exists yet**.

### Domain Knowledge

- Standard LSP practice: analyze the open file in the context of its compilation unit
  (here, the package = one directory), overlaying unsaved editor buffers on disk contents
  ("overlay" pattern, as in gopls). Publish diagnostics per-URI.
- `EnrichForeign` may shell out to `go list` for external modules (`foreign.go:495`);
  same-module imports resolve offline by walking to `go.mod` (`foreign.go:452`). The `go list`
  call is the only expensive step.
- Project is zero-dependency: tests use stdlib `testing` only (no testify). The `analyze`
  and `check` `foreign_test.go` files (with injected `DirResolver` / `testdata` fixtures) are
  the test templates.

## Decision Points

- [x] **D1 — How the LSP gets per-file results (PINNED)**: `runPackage(srcs, tables)` is
  unexported and returns `([][]Diagnostic, error)`. `check.AnalyzePackageInDir(srcs, dir)`
  exists (`check.go:169`) but hardcodes `nil` → `DefaultResolver` and discards
  `EnrichForeign`'s `[]error`, so it **cannot inject a fake resolver** (FR-008) and cannot
  log foreign-resolution failures (FR-005/FR-006). It also has two real callers
  (`cmd/goal/main.go`, `check/foreign_test.go`) whose calls must not break. **Decision**: add
  `func AnalyzePackageInDirWith(srcs []string, dir string, resolve analyze.DirResolver) ([][]Diagnostic, []error, error)`
  that composes `BuildPackage` + `EnrichForeign` (returning its `[]error` for logging) +
  `runPackage`; reimplement `AnalyzePackageInDir` to delegate with `resolve=nil` and discard
  the `[]error` (preserving its signature/callers). The LSP owns the buffer overlay (`srcs`);
  `check` owns the composition. Finalize the exact name in the plan step but keep this shape.
- [x] **D2 — File discovery for a directory**: prefer a cheap single-directory `*.goal` read
  over `project.Discover` (which walks the whole subtree) to avoid scanning the project on
  every debounced compile. Reuse Discover's package-name conflict rule for the fallback.
- [x] **D3 — Buffer overlay**: match package files to the directory; for each, if an open
  `doc` exists for its URI, use buffer text (unsaved wins), else disk text. Sort srcs by path
  for deterministic merge; remember the analyzed file's index.
- [x] **D4 — IO injection (PINNED)**: thread two seams through the server: a directory-file
  reader and an `analyze.DirResolver`. Keep `NewServer(out io.Writer)` intact (production
  defaults) so `cmd/goal/main.go:101` is unchanged; add a sibling constructor
  `NewServerWithIO(out io.Writer, files dirReader, resolve analyze.DirResolver) *Server`
  (chosen over functional options for symmetry with the existing single constructor and to
  keep the test wiring explicit). Define the reader seam as
  `type dirReader func(dir string) ([]fileSrc, error)` where `fileSrc` is `{path, src string}`
  (NOT `project.File`, to avoid a recursive Discover). The production `dirReader` lists
  `*.goal` in one directory and reads each.
- [x] **D6 — Path↔URI round-trip (PINNED, from audit M3)**: the `docs` map is keyed by URI,
  so overlay needs **path→URI** as well as URI→path. For each on-disk sibling path, the
  server computes its `file://` URI to probe `docs` (buffer wins over disk). Add both
  helpers: `uriToPath(uri string) (string, ok bool)` (accept `file:` only, percent-decode,
  strip empty authority/leading slash per platform) and `pathToURI(path string) string`.
- [x] **D7 — Sibling re-scheduling (PINNED, from audit C1/M4)**: `didChange`/`didOpen`/
  `didClose` for a file must schedule re-analysis of every *open* file in the same directory,
  not just the changed file. Because the package-level entry returns `[][]Diagnostic` for all
  srcs in one pass, a single analysis triggered by A's change can publish refreshed
  diagnostics for every open sibling — preferred over N separate `compileLatest` runs.
  Supersede keys remain per-URI (each published file keyed by its own latest version).
- [x] **D8 — Package-name conflict reuse**: `project.packageName` is unexported; only
  `project.PackageClause` is exported. The single-directory reader must re-check the
  one-package-per-dir rule via `PackageClause` (or export a helper) — do not assume a
  ready-made call. Conflict ⇒ single-file fallback (FR-005a).
- [x] **D9 — Analyzed file absent from disk listing (from audit M5)**: if the analyzed URI's
  path is not among the directory's disk files, append its buffer to `srcs` and record that
  index so its diagnostics are produced and attributed correctly.
- [ ] **D5 — Caching (OQ-1/OQ-2)**: start with no cache (rely on debounce). Add a
  directory-scoped enriched-tables cache only if a representative package is visibly laggy.

## Recommendations

1. Add a new exported composition in `internal/check` that the LSP calls with overlaid
   `srcs` + `dir` + injected resolver, returning per-file diagnostics (D1).
2. In `internal/lsp`, add: URI→path conversion, a single-directory `.goal` reader (injected),
   buffer overlay, analyzed-file index tracking, per-file publish, and a single-file fallback
   on any resolution/discovery failure (logged to stderr).
3. Inject IO seams via the server constructor with production defaults; fake them in tests
   (D4). Reuse the existing `DirResolver` seam for foreign resolution.
4. Defer caching until measured (D5); rely on the existing debounce + supersede first.
5. Test with stdlib `testing`: multi-URI same-package integration tests, a parity test vs
   `AnalyzePackageInDir`, and fallback tests (non-`file:` URI, package-name conflict).

## Risks / Watch-items

- `runPackage` unexported — settle the export surface in plan; don't duplicate the composition.
- URI→path correctness (percent-encoding, leading `/`; darwin primary). Cover `file:///abs/x.goal`.
- Buffer/disk skew: open file in a *different* directory must not be pulled in; match by dir, overlay by URI.
- Don't regress the synchronous test path (`file:///x.goal`, no real dir) — it must cleanly hit single-file fallback.
- Foreign resolution failures stay non-fatal (type deferred; other diagnostics intact).

## Sources

- `internal/lsp/{server.go,diagnostics.go,server_test.go}`
- `internal/check/{check.go,convert.go,fields.go,foreign_test.go,check_package_test.go}`
- `internal/analyze/{analyze.go,foreign.go,methods.go,foreign_test.go,analyze_test.go}`
- `internal/project/project.go`
- `cmd/goal/main.go`, `editors/vscode/src/extension.ts`
