# US-036 Lower and emit match including value position

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop policy)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Lower and emit goal `match` over an enum on the AST backend, in both statement
position and value position (`return match …`, `var name T = match …`), as a Go
type-switch over the §8.1 sum encoding. Add a new value-position-match example to
the corpus. The 02-match cases plus the new case must pass the behavioral tier
through the new (AST) backend.
