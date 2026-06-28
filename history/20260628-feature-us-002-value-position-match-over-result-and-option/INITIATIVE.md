# US-002 value-position match over Result and Option

**Type**: feature
**Created**: 2026-06-28
**Branch**: main (loop-runner: no branch, linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-28 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Lower a `match` over a Result or Option when its result is used as a value — a
return result or an assignment RHS — so a goal author can bind or return the
outcome of a match directly. Today value-position Result/Option match reaches the
generic expr path and fails with `unsupported expression *ast.MatchExpr`.
