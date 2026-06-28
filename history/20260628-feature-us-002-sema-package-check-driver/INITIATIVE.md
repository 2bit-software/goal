# US-002 sema package-check driver

**Type**: feature
**Created**: 2026-06-28
**Branch**: main (loop-runner: no branch — linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-28 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Add a package-level sema check entry point so multi-file goal packages produce
per-file diagnostics from the AST checker. Mirrors the legacy
check.AnalyzePackageInDir so US-004 is a near drop-in swap.
