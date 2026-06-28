# us-025-corpus-interpreter-runner

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop runs linear on the working branch; no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

US-025: Add a corpus interpreter runner. `internal/corpus` provides `RunInterp`:
it loads a corpus case, runs it through the goscript interpreter (internal/interp),
and compares observable behavior (doctest output / asserted result) the same way
the Go behavioral tier does. A unit test runs the doctest corpus cases through
`RunInterp` and asserts they pass.
