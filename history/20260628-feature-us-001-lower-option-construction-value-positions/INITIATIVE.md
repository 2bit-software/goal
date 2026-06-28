# US-001 Lower Option construction in value positions

**Type**: feature
**Created**: 2026-06-28
**Branch**: main (loop-runner: no branch, linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-28 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Option.Some/None must lower wherever an Option value is produced — not only at a
direct return or as a Result.Ok payload, but in var-assignment, call-argument,
struct-field, and slice/map-literal positions — so optional values compose anywhere.
