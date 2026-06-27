# US-050 LSP rename and find references

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs on the base branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Add `textDocument/references` and `textDocument/rename` to the goal LSP. Both
resolve through the AST symbol graph US-048 built for go-to-definition (and
US-049 reused for hover): parse the open buffer, index top-level declarations,
and walk references keyed by structural parent. Where go-to-definition maps a
reference to its declaration, references/rename invert the graph — given the
symbol under the cursor, return every occurrence (declaration name + all
reference sites), as `Location[]` for references and a `WorkspaceEdit` (one
`TextEdit` per occurrence) for rename. Single-document scope, matching the
earlier editor stories.
