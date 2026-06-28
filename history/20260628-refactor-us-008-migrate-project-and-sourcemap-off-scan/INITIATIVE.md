# US-008 Migrate project and sourcemap off scan

**Type**: refactor
**Created**: 2026-06-28
**Branch**: main (loop-runner: no branch, linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| scaffold | in_progress | 2026-06-28 |
| verify | pending | - |
| cutover | pending | - |
| cleanup | pending | - |
| done | pending | - |

## Description

Migrate internal/project and internal/pipeline/sourcemap so they import neither
the scan lexer (scan.Lex/Token/Match*) nor internal/analyze. Package-name
detection and the source map must be derived from parser.ParseFile / AST node
offsets (FuncDecl.Pos()/End() etc.). Existing project and source-map tests must
pass.
