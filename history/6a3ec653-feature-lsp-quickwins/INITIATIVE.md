# Initiative: lsp-quickwins

**Type**: feature
**Status**: complete
**Created**: 2026-06-26
**ID**: 6a3ec653-feature-lsp-quickwins

## Completion

**Completed**: 2026-06-26

### Outcomes
- Feature: High-ROI LSP quick wins — **Complete**
  1. "Idiomatize file (goal fix)" code action (`source.fixAll.goal`, version-pinned edit) wrapping `internal/fix.File`.
  2. Precise diagnostic ranges (token end offsets, line-end fallback).
  3. Document symbols / outline via a bounded two-phase token walk.

Binary rebuilt + installed to GOBIN; capabilities verified. No VSCode extension change.

### Notes
- Ran **automode, no branching**: committed directly to `main` (no branch/push/PR).
- 11 new tests; full `go test ./...` green; gofmt/vet clean.
- **User action**: reload the VSCode window to load the new server.
- Follow-ups: OQ-1 (unfixable `fix` reports as info diagnostics), OQ-2 (child symbols).

## Steps

| Step | Profile | Status | Updated |
|------|---------|--------|--------|
| spec | feature | complete | 2026-06-26 |
| plan | plan | complete | 2026-06-26 |
| tasks | tasks | complete | 2026-06-26 |
| implement | implement | complete | 2026-06-26 |

## Description

Add three high-ROI capabilities to the `goal` language server: (1) an "Idiomatize file"
code action wrapping `internal/fix.File` (version-pinned full-document WorkspaceEdit), (2)
precise diagnostic ranges from `scan.Lex` token end offsets, (3) document symbols (outline)
via a dedicated token-walk pass. All single-buffer; no cross-file. Running **automode, no
branching** — work lands on `main`.

## Goals

- Expose the existing, tested idiomatization engine as a one-click / on-save code action.
- Make diagnostic squiggles cover the offending token, not the whole line.
- Populate outline / breadcrumbs / ⌘⇧O via `documentSymbol`.
- No VSCode extension change (advertise capabilities; vscode-languageclient auto-registers).
- Rebuild + install the binary and reload VSCode on completion.

## Progress

- 2026-06-26 — **plan complete**. `implementation-plan.md` + `technical-spec.md` written
  (reuse-first; new code = LSP glue + a token walk). Plan audit run: confirmed lexer/grammar
  shapes and helper signatures; folded in the MAJOR fix that a bodyless `from`/`derive func`
  or `type X = …` alias must NOT use an unbounded forward `{`-scan (would swallow the next
  decl) — ranges are now bounded by the next top-level decl keyword, bodyless end via
  `NextNewline`. Also fixed `wantsKind` dead term, `i>0` guard, alias-via-`=`.
- 2026-06-26 — **spec complete**. Spec + research written; completeness & AI-consumer audits
  run. Folded in: version-pinned edit (avoid clobbering concurrent typing), `context.only`
  honoring, code-action title, build/install/reload as in-scope deploy, precise-range
  multi-line/fallback rules, and the correction that `scan.ScanFuncs` lacks receiver/keyword
  info so symbols use a dedicated token walk (`documentSymbols(src)` in symbols.go).
