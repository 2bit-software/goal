# US-001 foreign enrichment to sema.Info

**Type**: feature
**Created**: 2026-06-28
**Branch**: main (loop runs on base branch; no feature branch per loop policy)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-28 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Add a native (AST-driven, no token re-lex) foreign-type enrichment path to
internal/sema so the AST checker can resolve out-of-package struct fields and
method signatures without analyze.Tables. Mirrors analyze.EnrichForeign.
