# US-017 parse package imports declarations

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs linear on base)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Add internal/parser (Phase 1.4, declaration tier): a hand-written recursive-descent
parser that turns the lexer's token stream into an ast.File for the Go subset —
the package clause, imports (single/grouped/named/blank), and func/type/var/const
declarations. Statement bodies (US-018) and full Pratt expressions (US-019) are
deferred; function bodies are skipped as a balanced-brace BlockStmt for now.
