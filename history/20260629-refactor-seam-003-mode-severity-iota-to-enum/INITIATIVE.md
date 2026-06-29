# SEAM-003: sema.Mode and sema.Severity iota -> goal enum

**Type**: refactor
**Created**: 2026-06-29
**Branch**: main (no branch per loop-runner; linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-29 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Convert selfhost/sema Mode (ModeNone/ModeResult/ModeResultClosed/ModeOption) and
Severity (Error/Warning) from `type X int` + iota to goal `enum`s, and convert
every cross-package consumer atomically: ==/!=/switch over Mode/Severity become
`match`; variant references become `Enum.Variant` form; enum zero values are set
explicitly at every constructor (enum zero is nil, not the first variant).
Consumers span selfhost/sema, selfhost/backend (lower.goal, emit.goal), and
selfhost/typecheck. Gates: task check, task build, task fixpoint green; corpus
behavioral tier unchanged.
