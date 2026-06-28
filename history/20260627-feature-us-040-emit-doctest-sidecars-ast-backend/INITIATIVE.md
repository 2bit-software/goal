# US-040 Emit doctest sidecars on new path

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs on the working branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

The new AST backend must extract `///` doctests and emit the `_test.go` sidecar,
lowered through the same emit path as function bodies, so the 11-doctests cases
pass the doctest tier through the new (AST) engine.
