# US-002 define interp value model

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Define the uniform runtime value representation for the goscript tree-walking
interpreter (internal/interp). A single Value type covers int, float, string,
bool, nil, struct, slice, map, and function, plus a universal tagged-union
Variant{TypeID, Tag, Fields} used uniformly for enum / Result / Option — distinct
from the Go backend's optimizations (Result->(T,error), Option->*T).
