# SEAM-004: seal ast category interfaces + convert AST type-switches to match

**Type**: refactor
**Created**: 2026-06-29
**Branch**: main (loop-runner: no branch, linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-29 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Seal ast.Node/Expr/Stmt/Decl/Spec in selfhost/ast and convert the AST-family
type-switches to goal `match`, showcasing exhaustive match over the compiler's
central data structure. All CAP-3a/b/c/d prerequisites are proven.
