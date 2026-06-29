# US-012 idiomatic audit typecheck

**Type**: refactor
**Created**: 2026-06-29
**Branch**: main (loop runs on base branch for linear history; no feature branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-29 |
| plan | completed | 2026-06-29 |
| tasks | completed | 2026-06-29 |
| implement | completed | 2026-06-29 |
| verify | in_progress | 2026-06-29 |

## Description

Step 3 (idiomatic audit) of the SELF-HOST IDIOMATIC plan, package `selfhost/typecheck`.
Convert genuinely-fallible package-internal helpers to Result/Option + `?` where
behavior-preserving; convert internal switch-over-in-file-enum to `match` where it fits;
record genuine refusals-with-reason in DECISIONS.md. The US-003 verbatim selfhost is the
behavioral oracle: never change a public signature its tests pin; typecheck depth tests
must pass against the transpiled package; `task fixpoint` must stay byte-identical green.
