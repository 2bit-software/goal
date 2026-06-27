# US-029 Reimplement exhaustiveness check on AST

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs on the base rewrite branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Reimplement the feature-02 match exhaustiveness check over the new AST front-end
in `internal/sema`, replacing the token-scanning `internal/check/exhaustive.go`
implementation. A `match` over a known enum must cover every variant or supply an
explicit `_` rest-arm; a match over an enum not declared in the file defers with a
located Warning; matches on Result/Option are skipped (owned by their own features).
Every exhaustiveness-related case in `testdata/check/02-match` must pass through the
sema checker via the corpus check runner.
