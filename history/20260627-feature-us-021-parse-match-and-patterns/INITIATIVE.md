# us-021-parse-match-and-patterns

**Type**: feature
**Created**: 2026-06-26
**Branch**: ralph/ast-frontend-rewrite (loop runs on the base branch; no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | done | 2026-06-26 |
| plan | done | 2026-06-26 |
| tasks | done | 2026-06-26 |
| implement | done | 2026-06-26 |
| verify | in_progress | 2026-06-26 |

## Description

US-021: Parse match and patterns. The parser must parse `match` in both
statement position and value/expression position (`var x = match ...`,
`return match ...`), including variant patterns with payload bindings
(`Status.Active(a)`) and the rest pattern (`_`). A test asserts a
statement-position and a value-position match both parse with the expected arms.
