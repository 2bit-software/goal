# US-011 token package with positions

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Add `internal/token`: a token type carrying source positions so the future AST
can record real line/col locations. Defines `Kind` constants for every goal
lexeme and `Pos{Offset, Line, Col}`. A unit test asserts `Kind`/`String`
round-trips and `Pos` ordering.
