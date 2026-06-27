# US-037 Lower and emit closed-E Result

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop constraint)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | - |
| tasks | completed | - |
| implement | completed | - |
| verify | in_progress | - |

## Description

Lower and emit closed-E Result (a `Result[T, E]` whose E is not `error`, feature
06) on the AST backend: the §8.1 Ok/Err sum constructors, the closed-E `match`
type-switch, the closed-E `?` propagation with auto-invoked From-conversion, and
the generic Result prelude emitted once per file. The three 06-error-e cases must
pass the behavioral tier (temp-module go build + go vet) through the new (AST)
engine.
