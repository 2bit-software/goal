# US-030 Reimplement field-completeness check on AST

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop policy)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

sema implements the no-zero-value (spec §8) field-completeness check over the
AST: struct-literal and enum-variant-construction completeness, with
`...defaults`/`...derive(...)` spreads as complete-by-construction opt-outs,
unresolved literal types deferred as Warnings, and omissions as Errors. Every
field-completeness case in testdata/check/08-no-zero-value must pass through the
sema checker via the corpus runner.
