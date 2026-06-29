# US-004 Stand up bootstrap and fixpoint harness

**Type**: feature
**Created**: 2026-06-29
**Branch**: main (loop-runner: no branch — linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-29 |
| plan | completed | 2026-06-29 |
| tasks | completed | 2026-06-29 |
| implement | completed | 2026-06-29 |
| verify | in_progress | 2026-06-29 |

## Description

Stand up the 3-stage bootstrap and byte-identical fixpoint check as repeatable
Taskfile targets so every ported package can be validated against the trust
proof from the start. Add a `selfhost/` directory holding a thin `package main`
goal program (a `goal build --emit` equivalent), and Taskfile `bootstrap` and
`fixpoint` targets that run stage-0 -> goal-c-1 -> goal-c-2 and `diff -r` the Go
emitted by goal-c-1 and goal-c-2 for the compiler's own source. The fixpoint
target must exit 0 (byte-identical) against the current skeleton.
