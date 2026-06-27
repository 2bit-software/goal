# us-016-eval-option-tagged-union

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop working branch — no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

US-016: Eval Option as tagged union. The goscript interpreter (internal/interp)
must represent `Option.Some`/`Option.None` as the universal tagged-union Value —
no `*T` optimization — and match over them, binding the unwrapped Some payload.
