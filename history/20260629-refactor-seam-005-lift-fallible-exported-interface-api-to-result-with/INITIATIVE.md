# SEAM-005 lift fallible exported/interface API to Result with callers

**Type**: refactor
**Created**: 2026-06-29
**Branch**: main (loop-runner: no branch, linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| scaffold | in_progress | 2026-06-29 |
| verify | pending | - |
| cutover | pending | - |
| cleanup | pending | - |
| done | pending | - |

## Description

Lift the genuinely-propagating fallible exported/interface API of the self-hosted
goal compiler to Result/? together with cross-package callers and interface
contracts, while honestly carving out the sites blocked by SEMANTICS (not scope).
SEAM story under the relaxed gate: emitted Go may change, re-proven via task
fixpoint (stage1==stage2) + corpus behavioral tier + reviewed golden regen.
