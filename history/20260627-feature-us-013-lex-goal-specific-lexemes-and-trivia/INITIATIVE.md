# US-013 Lex goal-specific lexemes and trivia

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

Extend internal/lexer so the goal-specific lexemes are first-class single
tokens: `?` → QUESTION, `=>` → FAT_ARROW (one token, not `=` then `>`), and
`...` → ELLIPSIS. Retain `///` doc-comment content as DOC_COMMENT trivia
(distinct from an ordinary `//` COMMENT). The contextual keywords
implements/sealed/from/derive must continue to lex as IDENT (they are not
reserved in internal/token). The token kinds already exist (US-011); this story
wires the lexer (US-012) to emit them.
