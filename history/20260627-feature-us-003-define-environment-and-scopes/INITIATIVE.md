# US-003 Define environment and scopes

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop: no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

internal/interp Env supports Define, Lookup, and NewChild with inner scopes
shadowing outer scopes and lookups falling through to parents; Lookup of an
undefined name returns a not-found error.
