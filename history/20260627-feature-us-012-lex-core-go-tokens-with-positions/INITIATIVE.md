# US-012 lex core Go tokens with positions

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs linear on base)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

Add `internal/lexer`, a scanner that turns goal source into `[]token.Token` for
the core Go subset: identifiers, keywords, integer/float/imaginary/char/string
literals, and the full operator + delimiter set — each carrying an accurate
`token.Pos{Offset, Line, Col}`. Goal-specific lexemes (`?`, `=>`, `...`, `///`
doc comments, comment-trivia attachment) are explicitly deferred to US-013.
