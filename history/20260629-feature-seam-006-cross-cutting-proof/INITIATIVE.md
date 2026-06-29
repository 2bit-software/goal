# SEAM-006 Cross-cutting proof: the compiler showcases goal end-to-end

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

Whole-compiler proof that the SEAM conversions hold together: confirm fixpoint
+ corpus tiers green, tally per-seam what converted vs documented semantic
non-fit, run `goal fix` over the selfhost tree to confirm no residual
auto-convertible propagation, and quantify the idiom shift (type-switch->match,
iota->enum, fallible API->Result) measurably from the actual tree. Record the
META-finding: the deep idioms required four NEW compiler capabilities
(SEAM-CAP, CAP-2, CAP-3a-d) that did not exist before this PRD.
