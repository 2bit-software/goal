# US-001 corpus Case model and loader

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no branch created — loop runs on base)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Add `internal/corpus` defining a runner-independent `Case` model and a manifest
`Load(path)` loader, per Phase 0.1 of REWRITE-ARCHITECTURE.md, so the golden
suite is decoupled from package layout.
