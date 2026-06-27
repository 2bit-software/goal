# US-018 Eval implements dispatch

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop runs on the existing branch; no per-story branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

The goscript tree-walking interpreter (internal/interp) must honor `implements`:
a struct satisfying a sealed/ordinary interface is dispatched through the
interface, calling the correct concrete method at runtime. The acceptance proof
is a unit test over a 07-implements shape that calls an interface method on
differently-typed values and asserts each concrete implementation runs.
