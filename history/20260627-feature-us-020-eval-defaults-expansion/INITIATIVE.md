# US-020 Eval defaults expansion

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop runs on the existing base branch — no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

As a runtime author, I need `...defaults` so no-zero-value construction fills
fields at runtime. The goscript tree-walking interpreter (internal/interp) must
expand `...defaults` at struct construction, filling unset struct fields with
their safe zero values while preserving explicitly set fields.
