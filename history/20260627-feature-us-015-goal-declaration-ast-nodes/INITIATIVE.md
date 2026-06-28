# us-015-goal-declaration-ast-nodes

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs linear on base)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | done | 2026-06-27 |
| plan | done | 2026-06-27 |
| tasks | done | 2026-06-27 |
| implement | done | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

US-015: Add goal declaration AST nodes. The internal/ast package gains the
goal-specific declaration surface so enums and friends are first-class:
EnumDecl/Variant/PayloadField, SealedInterfaceDecl, ImplementsClause, and the
from/derive modifiers on FuncDecl. Walk descends into each new node's children,
proven by a unit test.
