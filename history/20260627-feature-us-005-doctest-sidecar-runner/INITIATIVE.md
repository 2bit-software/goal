# US-005 doctest sidecar runner

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop constraint)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Build the doctest sidecar runner for internal/corpus (Phase 0 of the AST
front-end rewrite). The runner compares pipeline Output.Test to the
.go.expected doctest sidecar (gofmt-normalized) for every doctest case in the
manifest, and a test runs all doctest cases against pipeline.Transpile.
