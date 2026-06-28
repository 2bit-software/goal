# us-014-eval-value-position-match

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop runs on the existing branch; no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | completed | 2026-06-27 |
| verify | pending | - |

## Description

US-014: The goscript tree-walking interpreter (internal/interp) must evaluate
value-position `match` — `return match`, `x := match`, and `var x = match` —
yielding the selected arm's value as an expression result.
