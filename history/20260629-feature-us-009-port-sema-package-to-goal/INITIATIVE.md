# US-009 Port sema package to goal

**Type**: feature
**Created**: 2026-06-29
**Branch**: main (loop-runner: no branch, linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-29 |
| plan | completed | 2026-06-29 |
| tasks | completed | 2026-06-29 |
| implement | completed | 2026-06-29 |
| verify | in_progress | 2026-06-29 |

## Description

Reimplement internal/sema as goal-native source under selfhost/sema so
resolution and checking are goal-native. It must transpile via the smoke gate
and the generated Go must compile (go/parser, go/format, go/types pass through
as foreign imports), and the existing sema tests must pass against the
transpiled package.
