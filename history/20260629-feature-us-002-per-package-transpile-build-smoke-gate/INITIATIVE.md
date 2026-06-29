# US-002: Per-package transpile-and-build smoke gate

**Type**: feature
**Created**: 2026-06-29
**Branch**: main (loop-runner: no branch, linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-29 |
| plan | completed | 2026-06-29 |
| tasks | completed | 2026-06-29 |
| implement | completed | 2026-06-29 |
| verify | in_progress | 2026-06-29 |

## Description

Add a gate that transpiles each in-scope compiler package through the goal
front-end and runs `go build` on the generated Go, so silent transpile defects
(which the checker does not flag) are caught. Covers at least token, lexer, ast,
parser, sema, project, pipeline, backend. Must be green after US-001 and fail if
any covered package transpiles to non-compiling Go.
