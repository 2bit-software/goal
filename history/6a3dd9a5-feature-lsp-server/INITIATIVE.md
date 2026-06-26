# Initiative: lsp-server

**Type**: feature
**Status**: in_progress
**Created**: 2026-06-25
**ID**: 6a3dd9a5-feature-lsp-server

## Steps

| Step | Profile | Status | Updated |
|------|---------|--------|--------|
| spec | feature | complete | 2026-06-25 |
| plan | plan | complete | 2026-06-25 |
| tasks | tasks | complete | 2026-06-25 |
| implement | implement | complete | 2026-06-25 |

## Description

Layer 2 editor integration, milestone 1: a Go-hosted Language Server that surfaces
Goal's existing lexical check violations as inline VS Code diagnostics (red/yellow
squiggles), wired to the Layer 1 extension via `vscode-languageclient`. Built on
`main` per "don't branch"; run in AutoMode with the full feature workflow.

## Goals

- Show guarantee violations inline in the editor on open/edit, no manual `goal check`.
- Reuse the in-memory `check.Analyze`/`AnalyzePackage` surface; map byte-offset
  diagnostics to LSP ranges in the original `.goal` source.
- Preserve the project's zero-dependency Go posture (hand-rolled JSON-RPC over stdio).

## Progress

- **spec**: business-spec.md, research-summary.md, technical-requirements-research.md
  written. Scope fixed to diagnostics-only, lexical checks, push model, `goal lsp`
  subcommand + thin VS Code client. Depth/type-backed checks and hover/etc. deferred.
- Branch note: the initiative tool again auto-created `feat/lsp-server`; honored
  "don't branch" — reverted to `main` (no swept commit this time) and deleted it.

