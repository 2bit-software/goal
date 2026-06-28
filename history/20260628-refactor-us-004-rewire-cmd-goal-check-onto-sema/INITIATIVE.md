# US-004 rewire cmd-goal check onto sema

**Type**: refactor
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

Rewire `cmd/goal` check onto the AST/sema checker so the legacy lexical
`internal/check` package is no longer on the live path. `cmd/goal/main.go`
`checkPackage` uses the US-002 sema package driver (`sema.AnalyzePackageInDir`)
and no longer imports `internal/check`. The typecheck depth stage and the
lexical/depth dedup (suppress-by-(basename,line,feature)) are preserved.
`check.OffsetToPosition` is dropped because sema diagnostics carry Line/Col
directly. An end-to-end test asserts `goal check` output over the corpus check
cases is unchanged from before the rewire.
