# US-008 eval control flow

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs on current branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | done | 2026-06-27 |
| plan | done | 2026-06-27 |
| tasks | done | 2026-06-27 |
| implement | done | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

Add control-flow statement evaluation to the goscript tree-walking interpreter
(internal/interp): three-clause and condition-only for loops, switch with cases
and a default clause, nested block scoping, and break/continue. if/else is
already implemented (US-004/US-006 seam).
