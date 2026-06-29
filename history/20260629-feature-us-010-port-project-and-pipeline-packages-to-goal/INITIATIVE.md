# US-010 port project and pipeline packages to goal

**Type**: feature
**Created**: 2026-06-29
**Branch**: main (loop-runner: no branch, linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-29 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Port internal/project and internal/pipeline to goal-native source under
selfhost/project and selfhost/pipeline so discovery and output types are
goal-native. Both must transpile via the US-002 smoke gate (BuildTranspiled)
to compiling Go, and the existing project and pipeline tests must pass against
the transpiled packages (BuildAndTest).
