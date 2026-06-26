# Implementation Plan: Cross-file & cross-package completeness in the LSP

Ordered, dependency-aware steps. Each step ends compilable. Tests are stdlib `testing`
(no testify — project is zero-dependency). See `technical-spec.md` for code shapes and
`research.md` for the pinned design decisions (D1–D9).

## Reuse posture

This feature is reuse-first by construction. It introduces **no new analysis** — it routes
the editor through machinery that already exists and is exercised by the CLI:

| Planned need | Reused existing code | New? |
|---|---|---|
| Cross-file table merge | `analyze.BuildPackage` + `Tables.Merge` | reuse |
| Cross-package Go enrichment | `analyze.EnrichForeign` | reuse |
| Import-path resolution seam | `analyze.DirResolver` / `DefaultResolver` | reuse |
| Per-file diagnostics | `check.runPackage` | reuse |
| Package-name clause read | `project.PackageClause`, `project.Ext` | reuse |
| Single-file fallback | `check.Analyze` (current behavior) | reuse |
| Resolver-injectable entry | `check.AnalyzePackageInDirWith` | **new (thin wrapper)** |
| URI→path | `lsp.uriToPath` | **new** |
| Dir file reader seam + prod impl | `lsp.dirReader` / `osDirReader` | **new** |
| Package-aware compile + publish | rewrite of `lsp.compile` | **new wiring** |

`AnalyzePackageInDir` is NOT duplicated — it is reimplemented to delegate to the new
resolver-injectable variant, so its signature and its two existing callers
(`cmd/goal/main.go`, `internal/check/foreign_test.go`) are untouched.

## Steps

### Step 1 — `check.AnalyzePackageInDirWith` (foundation, no behavior change yet)
- Add `AnalyzePackageInDirWith(srcs, dir, resolve) ([][]Diagnostic, []error, error)` composing
  `BuildPackage` + `EnrichForeign(resolve)` + `runPackage`.
- Reimplement `AnalyzePackageInDir` to delegate with `resolve=nil`, discarding `[]error`.
- **Test**: existing `internal/check/foreign_test.go` still passes; add a test that
  `AnalyzePackageInDirWith` with an injected fake resolver resolves a foreign type and
  surfaces an empty/non-empty `[]error` appropriately.
- *Depends on*: nothing. *Risk*: low (pure refactor + addition).

### Step 2 — `lsp.uriToPath` + `uri_test.go`
- Add `internal/lsp/uri.go` with `uriToPath` (FR-003b/D6).
- **Test**: `file:///a/b.goal` → `/a/b.goal`; `file:///a/p%20q/x.goal` → `/a/p q/x.goal`;
  `untitled:Untitled-1` and `http://…` → `ok=false`.
- *Depends on*: nothing. *Risk*: low.

### Step 3 — IO seams + constructor in `server.go`
- Add `fileSrc`, `dirReader`, server fields `files`/`resolve`, `NewServerWithIO`, and rewire
  `NewServer` to call it with `osDirReader` + `analyze.DefaultResolver` (D4/FR-008).
- Add `osDirReader` (non-recursive single-dir `.goal` read using `project.Ext`).
- **Test**: `NewServer` wiring compiles; `cmd/goal/main.go:101` unchanged and still builds.
- *Depends on*: nothing. *Risk*: low. Confirm no import cycle (`lsp`→`project` is new; verify
  `project` does not import `lsp` — it does not).

### Step 4 — Package-aware compile in `diagnostics.go`
- Extract today's `compile` body into `compileSingle(uri, text, version)` (unchanged
  behavior, the fallback for non-`file:` URI / unreadable dir / package conflict only —
  NOT for a resolvable dir with no siblings, which is a package-of-one; see tech-spec §6).
- Implement `compile` per technical-spec §4: URI→path → dir → `s.files(dir)` → open-file
  snapshot → merge-by-path overlay (buffer wins, D3/FR-003) → analyzed-file index incl.
  unsaved-not-on-disk append (D9) → package-name conflict guard (FR-005a/D8) →
  `check.AnalyzePackageInDirWith(srcs, dir, s.resolve)` → log `[]error` (FR-006) →
  publish per open file in dir, each to its own URI/version, superseded-guarded
  (FR-004/FR-010/D7).
- **Serialize analysis (audit MAJOR #2)**: add `analysisMu sync.Mutex`; hold it across the
  analysis path and re-snapshot `s.docs` after acquiring it, so the newest run publishes
  last. Release `s.mu` before the slow work (FR-009).
- Add helpers `openFilesInDir`, `conflictingPackageNames`, and the merge/sort.
- **didClose symmetry (audit #5b)**: after clearing the closed file's diagnostics, `schedule`
  re-analysis of the remaining open files in its directory.
- *Depends on*: Steps 1–3. *Risk*: medium — the core wiring; cover with the tests in Step 5.

### Step 5 — LSP integration + parity + fallback tests
- Build a fake `dirReader` (in-memory map dir→files) and a fake `DirResolver` (maps a test
  import path to a fixture Go pkg dir) test harness; reuse the synchronous-server pattern
  (`debounce<=0`) and the published-message capture from `server_test.go`.
- **Migrate the existing `TestServerPublishesDiagnosticsOnOpen` to `NewServerWithIO`** with a
  fake `dirReader` (no siblings) + no-op resolver, so it no longer reads the real `/`
  (audit MAJOR #4). Add a regression test: `didClose` of one file re-publishes its open
  sibling (audit #5b).
- Tests, one observable behavior each:
  1. Cross-file: `derive`/literal targeting a sibling-declared struct no longer defers; a
     genuinely unsourced field is an Error (FR-001).
  2. Foreign: `derive`/`from`/`?` into an imported Go type resolves via the fake resolver (FR-002).
  3. Unsaved A→B refresh: editing A republishes B with the reference now resolved (FR-003/FR-010).
  4. Attribution: each file's diagnostics publish to its own URI only (FR-004).
  5. Non-`file:` URI → single-file fallback, still publishes (FR-005).
  6. Package-name conflict in dir → single-file fallback + stderr log (FR-005a).
  7. Open buffer in a different dir is excluded from the package view (FR-003a).
  8. Debounce coalesces; superseded stale publishes dropped (FR-009).
  9. Parity: LSP package path == `check.AnalyzePackageInDir` for a fixture package (FR-007).
- *Depends on*: Step 4. *Risk*: medium — fixture wiring; mitigated by reusing existing fixtures.

### Step 6 — Verify & polish
- `go build ./...`, `go vet ./...`, `go test ./internal/lsp/... ./internal/check/... ./internal/analyze/... -count=1`.
- Manual sanity: open the VSCode extension against a multi-file package, confirm the
  `completeness deferred` flood is gone for resolvable types (or document if not manually run).
- Update `DECISIONS.md` if the project records design decisions there (the repo has one).

## Sequencing rationale
Steps 1–3 are independent, low-risk foundations and can land in any order; Step 4 depends on
all three; Step 5 depends on Step 4. Each step compiles and keeps existing tests green.

## Open items carried into tasks
- Final method/helper names (cosmetic) — settle during tasks.
- D5 caching deferred unless Step 6 manual check shows lag (OQ-1/OQ-2).
- Whether to add a `DECISIONS.md` entry (follow repo convention).
