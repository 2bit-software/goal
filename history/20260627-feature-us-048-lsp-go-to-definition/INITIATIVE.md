# US-048 LSP go-to-definition

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no feature branch — loop runs on base)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | pending | - |

## Description

Add `textDocument/definition` to the goal LSP. A referenced symbol under the
cursor resolves to its declaration position via the AST symbol graph. The
acceptance criterion requires definition of a function call and an enum variant
to resolve to their declaration positions.
