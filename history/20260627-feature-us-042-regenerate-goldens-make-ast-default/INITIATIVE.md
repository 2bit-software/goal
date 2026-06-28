# US-042 Regenerate goldens and make AST default

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop constraint: no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Regenerate the exact .go.expected goldens from the new AST backend
(backend.Transpile) and make the AST engine the default, so the new front-end
path is canonical. The old splice engine remains available behind
`--engine=splice` for one release.
