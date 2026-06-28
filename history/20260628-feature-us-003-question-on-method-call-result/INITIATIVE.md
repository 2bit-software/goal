# US-003 — Allow `?` on method calls returning Result

**Type**: feature
**Created**: 2026-06-28
**Branch**: main (loop runs on base branch, no feature branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-28 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

`?` must work on a method call whose method returns a `Result` (or a trailing
`error`), so a goal author need not wrap every interface/method call in a plain
helper function first. The checker must accept `v := recv.M()?` where `M`
returns `Result[T, error]` with no `question-callee-no-error` diagnostic;
transpiling must bind the value and propagate the trailing error; a method
returning only `error` lowers to the single-variable `if err := recv.M(); err
!= nil` form; a method whose return does not end in `error` is still rejected.
