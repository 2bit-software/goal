# us-013-eval-statement-position-match

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop runs on current branch, no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

US-013: Eval statement-position match. The goscript tree-walking interpreter
(internal/interp) must evaluate statement-position match: dispatch on the
variant tag, bind the matched payload into scope, run the selected arm, and
panic 'unreachable' on the defensive default of a proven-exhaustive match.
