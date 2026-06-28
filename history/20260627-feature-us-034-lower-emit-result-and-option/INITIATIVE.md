# US-034 Lower and emit Result and Option

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs on base)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

As a goal developer, I need open-E Result and Option lowering on the new AST Go
backend so the core types work. lower+backend must produce native `(T, error)`
named returns for open-E `Result[T, error]` and `*T` for `Option[T]`, including
statement-position `match` over them. The 03-result and 04-option transpile cases
must pass the behavioral tier through the new (AST) engine.
