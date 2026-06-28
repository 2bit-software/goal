# US-007 migrate typecheck off analyze and scan

**Type**: refactor
**Created**: 2026-06-28
**Branch**: (none — loop runs on base branch `main` for linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| scaffold | in_progress | 2026-06-28 |
| verify | pending | - |
| cutover | pending | - |
| cleanup | pending | - |
| done | pending | - |

## Description

Migrate internal/typecheck off internal/analyze and the internal/scan token
lexer so the depth checker builds its function/signature/struct/sealed facts
from sema.ResolvePackage / sema.Info instead of analyze.Tables, and locates
`implements` clauses by walking the goal AST instead of scan.Lex.
