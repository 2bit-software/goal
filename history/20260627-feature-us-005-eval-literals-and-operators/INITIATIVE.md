# US-005 eval literals and operators

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop convention)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

Add expression evaluation to the goscript tree-walking interpreter
(internal/interp) so ordinary computation runs under interpretation: int/float/
string/bool literals plus arithmetic, comparison, logical (with short-circuit),
and unary operators, all with Go semantics.
