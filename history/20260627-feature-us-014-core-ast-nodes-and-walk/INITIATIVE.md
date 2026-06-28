# us-014-core-ast-nodes-and-walk

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop: no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

US-014: Define core AST nodes and Walk. As a compiler engineer, I need AST node
types for the Go subset plus a visitor so tools can traverse one tree.
internal/ast defines File, the Go decl/stmt/expr/type nodes goal uses, and
Walk(Visitor, Node). A test builds a tree by hand and asserts Walk visits every
node exactly once.
