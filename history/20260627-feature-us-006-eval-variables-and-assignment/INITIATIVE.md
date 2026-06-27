# US-006 Eval variables and assignment

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop working branch; no new branch per loop-runner)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Extend the goscript tree-walking interpreter (internal/interp) to evaluate
variable declarations and assignment so program state evolves during
evaluation: `var` declarations, short-var `:=`, `const`, and plain (`=`) plus
compound (`+=`, `-=`, etc.) assignment, reading and writing through Env.
