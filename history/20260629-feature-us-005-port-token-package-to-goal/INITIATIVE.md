# US-005 Port token package to goal

**Type**: feature
**Created**: 2026-06-29
**Branch**: main (loop runs on linear history, no branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | done | 2026-06-29 |
| plan | done | 2026-06-29 |
| tasks | done | 2026-06-29 |
| implement | done | 2026-06-29 |
| verify | in_progress | 2026-06-29 |

## Description

Reimplement internal/token as goal source under selfhost/token so the leaf of
the compiler dependency DAG is goal-native. The ported package must transpile
through the goal front-end (US-002 smoke gate), the generated Go must compile,
and the existing internal/token tests must pass against the transpiled package.
