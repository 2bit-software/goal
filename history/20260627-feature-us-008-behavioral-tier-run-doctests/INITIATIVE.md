# US-008 Behavioral Tier: Run Doctests

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop constraint)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Add a behavioral conformance tier that executes doctest sidecars. For each
KindDoctest case, transpile the input, write both the main Go output and the
emitted _test.go sidecar into an isolated temp module, and run `go test` on it,
so doctest behavior is proven (executed), not merely compiled. A test asserts
every doctest-bearing case passes go test in its temp module.
