# US-010 Eval builtins and methods

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop convention)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Add the `len`, `append`, `make`, and `panic` builtins to the goscript
tree-walking interpreter (internal/interp), and dispatch both value-receiver
and pointer-receiver methods declared on goal types.
