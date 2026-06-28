# us-035-lower-emit-question-real-gensym

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop constraint: no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

US-035: Lower and emit `?` with real gensym. Lower postfix `?` propagation in the
AST backend (internal/backend) using scope-aware generated identifiers instead of
the literal `__goal_` prefix, so the 05-question-prop cases pass the behavioral
tier and the generated Go contains no `__goal_` substring.
