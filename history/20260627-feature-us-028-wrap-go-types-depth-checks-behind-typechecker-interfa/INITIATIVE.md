# US-028 wrap go/types depth checks behind TypeChecker interface

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no feature branch — loop runs on the base branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Define a `TypeChecker` interface in `internal/typecheck` so the existing
transpile-then-go/types depth-checker (implements / must-use / no-zero-value
checks over the lowered Go) implements it, and the `go/types`-over-lowered-Go
crutch can be swapped for a native goal checker later without caller changes
(REWRITE-ARCHITECTURE.md §3.2, decision 4).
