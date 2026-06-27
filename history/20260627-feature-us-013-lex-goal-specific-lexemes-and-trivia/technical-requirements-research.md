# Technical Requirements / Research — US-013

## Existing seams

- `internal/token` already defines the target kinds: QUESTION, FAT_ARROW,
  ELLIPSIS, DOC_COMMENT (added in US-011). No token-package change needed.
- `internal/lexer` (US-012) currently defers these: `?` → ILLEGAL, `=>` → `=`
  then `>`, `...` → three PERIODs, `///` → ordinary `//` COMMENT. This story
  edits `scanOperator` and the comment path.

## Implementation hints

- In `scanOperator`:
  - Add a `'?'` case returning a one-rune QUESTION token.
  - In the `'='` case, longest-match `=>` to FAT_ARROW before falling back to
    the existing `=`/`==` handling.
  - In the `'.'` case, longest-match `...` to ELLIPSIS (peek two ahead) before
    the existing single PERIOD. Guard the existing `.`-then-digit FLOAT path in
    Next() — `...` has no digit after the dots so it is unaffected.
  - In the `'/'` case, detect `///` (a third `/`) and route to a DOC_COMMENT
    scan; keep `//` → COMMENT and `/* */` → COMMENT.
- DOC_COMMENT scanning mirrors scanLineComment but tags the token DOC_COMMENT;
  the Lit retains the full `/// ...` text (trivia content kept verbatim).
- Contextual keywords already lex as IDENT via token.Lookup (they are not in
  the keyword map) — assert this in the test, no code change.

## Test plan

- Extend `internal/lexer/lexer_test.go` with table tests asserting each lexeme
  yields exactly one token of the expected kind (count == 2 incl. EOF), plus a
  trivia test that `///` is DOC_COMMENT while `//` is COMMENT, and an
  identifier test for implements/sealed/from/derive.

No third-party tools or dependencies required.
