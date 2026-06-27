# US-012 eval enum construction

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no feature branch — loop runs linear history on the working branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Teach the goscript interpreter to construct enum variant values — both data-less
(`Status.Pending`) and payload-carrying (`Status.Active(since: now())`) — into the
universal tagged-union Value, and to read their payload fields. Sum types must
exist at runtime as tagged unions, distinct from the Go backend's optimizations.
