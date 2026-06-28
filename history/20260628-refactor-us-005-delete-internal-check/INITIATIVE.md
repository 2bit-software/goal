# us-005-delete-internal-check

**Type**: refactor
**Created**: 2026-06-28
**Branch**: (none — linear history on main, per loop-runner)

## Status

| Step | Status | Updated |
|------|--------|---------|
| scaffold | in_progress | 2026-06-28 |
| verify | pending | - |
| cutover | pending | - |
| cleanup | pending | - |
| done | pending | - |

## Description

US-005 — Delete internal/check package. Remove the legacy lexical checker now
that `goal check` runs on sema (US-004). Migrate every remaining live consumer
(cmd/goalc, internal/guide, internal/lsp, internal/typecheck, internal/corpus)
off internal/check, then delete the directory. No live import of internal/check
may remain outside attic/ and features/_cut/. Repoint corpus/check_runner.go and
its tests to the sema checker and remove the legacy CheckerFunc adapter.
