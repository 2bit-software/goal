# US-017 Eval question-mark unwinding

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop convention: no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

Implement postfix `?` (ast.UnwrapExpr) in the goscript tree-walking interpreter
(internal/interp) as non-local early return: on Err/None it unwinds to the
enclosing function's error/none return; on Ok/Some it yields the unwrapped
value; closed-E `from` conversions are applied during propagation.
