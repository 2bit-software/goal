# us-002-rename-typecheck-package-to-selfhost

**Type**: refactor
**Created**: 2026-06-29
**Branch**: main (loop runs on a linear history; no branch per loop-runner skill)

## Status

| Step | Status | Updated |
|------|--------|---------|
| scaffold | in_progress | 2026-06-29 |
| verify | pending | - |
| cutover | pending | - |
| cleanup | pending | - |
| done | pending | - |

## Description

US-002: mirror internal/typecheck/*.go into selfhost/typecheck/*.goal as a
VERBATIM rename (final package of the self-host rename step). Fix only
reserved-word identifier collisions; go/types, go/importer, go/ast, go/parser
pass through as foreign imports. Add a port_test that exercises the compile gate
(BuildTranspiled over the full dep closure) and the behavioral gate
(BuildAndTest running the fixture-free typecheck tests). No idiomatization —
that is US-004 (autofix) and US-012 (audit).
