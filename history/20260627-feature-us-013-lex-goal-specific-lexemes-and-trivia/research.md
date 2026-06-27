# Research — US-013

## Summary

No external research required. This is a self-contained extension of the
in-repo hand-written lexer (`internal/lexer`) to emit token kinds that already
exist in `internal/token` (US-011). The reference model is Go's own
`go/scanner` longest-match approach, already mirrored by the existing US-012
code.

## Findings

1. **Token kinds already present.** `internal/token/token.go` defines QUESTION
   (`?`), FAT_ARROW (`=>`), ELLIPSIS (`...`), and DOC_COMMENT (`///`) with
   correct kindNames entries. No token change needed.

2. **Contextual keywords already IDENT.** `token.keywords` deliberately omits
   implements/sealed/from/derive, and `scanIdentifier` only promotes a word to
   a keyword when `token.Lookup(...).IsKeyword()` is true. So they already lex
   as IDENT — the story only needs a test asserting this.

3. **Longest-match ordering (the gotcha).**
   - `///` must be matched before `//`: detect a third `/` after `//`.
   - `=>` must be matched before `=`/`==`: peek for `>` right after `=`.
   - `...` must be matched before a single `.`: the `.`-then-digit FLOAT branch
     in `Next()` is unaffected because `...` has no digit after the first dot,
     so `..` (two dots, no digit) still routes to `scanOperator`.

4. **Trivia semantics.** DOC_COMMENT is retained (emitted as a token with its
   full `/// ...` text as Lit), exactly like COMMENT is today — the lexer keeps
   comments in-stream; "attach as trivia" at the parser layer is a later story.
   For this story, "retained as trivia" = emitted as a distinct DOC_COMMENT
   token rather than swallowed or mis-typed as COMMENT.

## Confidence

High — purely local, fully covered by the existing test harness pattern.

## Recommended next steps

Edit `scanOperator` (the `/`, `=`, `.` cases) and add a `?` case plus a
`scanDocComment` helper; extend `lexer_test.go`.
