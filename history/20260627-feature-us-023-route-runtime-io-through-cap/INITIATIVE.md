# US-023 Route runtime IO through cap

**Type**: feature
**Created**: 2026-06-28
**Branch**: ralph/ast-frontend-rewrite (loop base; no new branch per loop-runner)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | complete | 2026-06-28 |
| plan | complete | 2026-06-28 |
| tasks | complete | 2026-06-28 |
| implement | complete | 2026-06-28 |
| verify | in_progress | 2026-06-28 |

## Description

Route the goscript interpreter's host effects (stdout writes from print/fmt,
and the future time/env reads) through a cap.CapabilitySet so authority is
explicit. The default Run path grants everything (cap.GrantAll()). A unit test
runs a printing program under GrantAll and captures the produced stdout through
the capability-mediated sink.
