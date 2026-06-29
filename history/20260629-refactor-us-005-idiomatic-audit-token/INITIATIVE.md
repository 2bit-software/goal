# US-005 Idiomatic audit: token

**Type**: refactor
**Created**: 2026-06-29
**Branch**: main (loop policy: no branch, linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-29 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Step 3 of 3 (idiomatic audit), first package: selfhost/token. Hand-upgrade
selfhost/token to goal idioms where they fit. The verbatim selfhost (US-003)
is the behavioral oracle: token tests must still pass against the transpiled
package and `task fixpoint` must stay byte-identical green.
