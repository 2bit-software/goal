# us-049-lsp-hover-with-types

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop runs on the existing branch; no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

US-049: Add LSP hover with types. As an editor user, I need hover so I can see a
symbol's type and doc. textDocument/hover returns the type and any doc comment
for the symbol under the cursor, derived from the AST symbol graph introduced in
US-048. A test asserts hover over a Result-returning function reports its
signature.
