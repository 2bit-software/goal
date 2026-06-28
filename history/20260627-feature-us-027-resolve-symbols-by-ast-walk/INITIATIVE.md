# us-027-resolve-symbols-by-ast-walk

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no feature branch — loop runs on base, linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

US-027: Resolve symbols by AST walk. internal/sema builds name-keyed facts
(enums, structs, function signatures, the from-registry, methods) by walking the
goal AST, replacing analyze's flat token scanning. A test asserts sema resolves
the same symbols as analyze.Build for representative inputs, including a struct
field whose type contains an embedded comma — the analyze.parseStructBody
whitespace-comma-split bug, which the structural AST walk fixes.
