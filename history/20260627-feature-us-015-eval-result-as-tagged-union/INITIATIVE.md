# us-015-eval-result-as-tagged-union

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs linear on the base branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

US-015: Eval Result as tagged union. The goscript interpreter (internal/interp)
represents Result.Ok / Result.Err as universal tagged-union Values — uniformly
for both open-E (`Result[T, error]`) and closed-E (`Result[T, SomeEnum]`), with
no `(T, error)` optimization — and matches over them, binding the Ok payload and
the Err error.
