# US-038 Lower and emit defaults and assert

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — PRD loop runs linear)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Lower `...defaults` (feature 08, §8.5) and `assert` (feature 10, §4.3/§8.6) on the
new AST backend so those constructs work through the `--engine=ast` path. The
08-no-zero-value and 10-assert corpus cases must pass the behavioral tier through
the new backend.
