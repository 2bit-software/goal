# US-013 Final idiomatic sweep and self-host proof

**Type**: feature
**Created**: 2026-06-29
**Branch**: main (loop runs on base branch — no feature branch per loop-runner)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-29 |
| plan | completed | 2026-06-29 |
| tasks | completed | 2026-06-29 |
| implement | completed | 2026-06-29 |
| verify | in_progress | 2026-06-29 |

## Description

Whole-compiler proof that no auto-convertible plain-Go propagation patterns remain
in the selfhost tree and that the idiomatic compiler still self-hosts. Run `goal fix`
across the entire selfhost tree (must report zero auto-convertible sites), record any
remaining deliberately-Go constructs in DECISIONS.md, prove `task fixpoint`
(goal-c-1 and goal-c-2 byte-identical), and confirm the goal-built compiler passes the
full corpus transpile + behavioral + check tiers.
