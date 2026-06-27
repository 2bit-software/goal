# US-004 Interpreter entry over AST and sema

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop convention)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Expose an interpreter constructor taking a parsed *ast.File (or package) plus
*sema.Info and a Run entry that executes func main, proving the goscript
interpreter is a back-end over the shared AST + sema front-end rather than the
Go-lowered output.
