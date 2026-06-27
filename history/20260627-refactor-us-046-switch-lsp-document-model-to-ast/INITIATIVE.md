# US-046 Switch LSP document model to AST

**Type**: refactor
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

Switch the LSP document model to the AST. internal/lsp document symbols (and
diagnostics) must be derived from the AST rather than the scan token walk; the
`scanDecls` token walk in lsp/symbols.go is removed. The existing LSP test
suite must pass unchanged in observable behavior.
