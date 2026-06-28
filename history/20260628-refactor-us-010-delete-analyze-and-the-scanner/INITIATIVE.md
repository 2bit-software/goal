# us-010-delete-analyze-and-the-scanner

**Type**: refactor
**Created**: 2026-06-28
**Branch**: main (loop-runner: no branch — linear history on base)

## Status

| Step | Status | Updated |
|------|--------|---------|
| scaffold | in_progress | 2026-06-28 |
| verify | pending | - |
| cutover | pending | - |
| cleanup | pending | - |
| done | pending | - |

## Description

Remove internal/analyze and the text/scanner-based lexer (internal/scan) so the
last lexer crutch is gone before self-host. Repoint the remaining live consumers
and the sema differential parity tests off analyze, then delete both packages.
