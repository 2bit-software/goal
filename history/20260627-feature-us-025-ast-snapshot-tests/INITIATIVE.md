# us-025-ast-snapshot-tests

**Type**: feature
**Created**: 2026-06-26
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs on the base branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-26 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

US-025: Add AST snapshot tests. Give the parser a way to render an AST to a
deterministic textual form (s-expression) and pin checked-in snapshot goldens
for one representative input per goal construct. A test compares the rendered
AST of each representative input to its checked-in snapshot, so parser
structure is pinned per construct (replaces the tree-sitter differential as the
loop-ready gate).
