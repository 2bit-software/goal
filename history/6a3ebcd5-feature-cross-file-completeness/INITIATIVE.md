# Initiative: cross-file-completeness

**Type**: feature
**Status**: complete
**Created**: 2026-06-26
**ID**: 6a3ebcd5-feature-cross-file-completeness

## Completion

**Completed**: 2026-06-26

### Outcomes
- Feature: cross-file & cross-package completeness in the LSP — **Complete**

The `goal` language server now analyzes an open `.goal` file in the context of its whole
package directory and imported Go packages, reusing `analyze.BuildPackage` +
`analyze.EnrichForeign` via a new resolver-injectable `check.AnalyzePackageInDirWith`. This
removes the `completeness deferred` flood in VSCode for types declared in sibling files or in
non-goal Go imports, with a single-file fallback when no package directory resolves.

### Notes
- Ran in **automode, no branching**: committed directly to `main`; no branch, push, or PR.
- 11 new tests; full `go test ./...` green; gofmt/vet clean.
- D5 caching deferred; symlinked package dirs a known limitation.

## Steps

| Step | Profile | Status | Updated |
|------|---------|--------|--------|
| spec | feature | complete | 2026-06-26 |
| plan | plan | complete | 2026-06-26 |
| tasks | tasks | complete | 2026-06-26 |
| implement | implement | complete | 2026-06-26 |

## Description

Make the `goal` language server (`goal lsp`, used by VSCode) analyze an open `.goal`
file in the context of its whole package directory and imported Go packages, instead of
single-file. This stops the existing completeness checks (feature 12 derive-totality,
feature 08 field-completeness, etc.) from emitting spurious `completeness deferred`
Warnings for types declared in sibling files or in non-goal (Go-only) imports.

Running mode: **automode, no branching** — all work lands on `main`.

## Goals

- Cross-file goal type resolution in the editor (sibling `.goal` files).
- Cross-package resolution into plain-Go imports (parity with `goal check`).
- Respect unsaved buffers; refresh open siblings when a file is fixed.
- Inject IO seams (dir reader + `DirResolver`) for stdlib-only testing.
- Graceful single-file fallback when no package directory resolves.

## Progress

- 2026-06-26 — **implement complete**. All 8 tasks done. `check.AnalyzePackageInDirWith`
  (resolver-injectable) + LSP package-aware `compile` (buffer overlay, sibling fan-out,
  `analysisMu` serialization, single-file fallback, didClose refresh). New: `lsp/uri.go`,
  `lsp/package_test.go`, `lsp/uri_test.go`. 11 new tests; full `go test ./...` green; gofmt/vet clean.
- 2026-06-26 — **plan complete**. `implementation-plan.md` + `technical-spec.md` written
  (reuse-first: no new analysis, only LSP wiring + 2 IO seams + 1 resolver-injectable check
  entry). Plan audit run: no import cycle; folded in 3 fixes — serialize package analysis
  with re-snapshot (concurrency race on sibling fan-out), corrected the `file:///x.goal`
  fallback assumption + migrate existing test to `NewServerWithIO`, and didClose must
  re-analyze remaining open siblings.
- 2026-06-26 — **spec complete**. Root cause: `internal/lsp/diagnostics.go:12` calls the
  single-file `check.Analyze`; the cross-file/cross-package machinery already exists
  (`BuildPackage` + `EnrichForeign`, shipped in `check.AnalyzePackageInDir`) but the LSP
  never uses it. Spec + research written; completeness & AI-consumer audits run; all
  CRITICAL/MAJOR findings folded in (sibling re-analysis fan-out, unsaved-file-in-dir
  boundary, resolver-injectable check entry, path↔URI seam, package-conflict fallback).
