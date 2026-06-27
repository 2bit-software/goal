# US-047 Add LSP semantic tokens

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop runs on the existing branch; no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

As an editor user, I need semantic tokens so goal constructs are highlighted by
role. The server must advertise and serve `textDocument/semanticTokens`
classified from the AST, and a test must assert the token classification of a
sample containing an enum, a match, and a `?` expression.
