# US-007 behavioral tier compile output

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop constraint)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | done | 2026-06-27 |
| plan | done | 2026-06-27 |
| tasks | done | 2026-06-27 |
| implement | done | 2026-06-27 |
| verify | pending | - |

## Description

For every transpile case the corpus runner writes Output.Go to a temp module
and runs `go build` + `go vet` on it. A test asserts every transpile case's
generated Go builds and vets cleanly. This behavioral tier is
implementation-independent and gates all later phases.
