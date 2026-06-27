# Technical Requirements / Research — US-012

## Foundations already in place

- `internal/token` (US-011) defines every `Kind` (literals, operators,
  delimiters, Go + goal keywords), `Pos{Offset, Line, Col}`, `Token{Kind, Lit,
  Pos}`, and `Lookup`/`IsKeyword`. The lexer emits `[]token.Token` using these.

## Design hints

- Model on `go/scanner` but smaller: a byte-offset cursor over the source with
  a running line/col tracker. Positions are first-class (the splice `scan.Lex`
  threw byte offsets away every pass — that is what this replaces).
- Longest-match operators: `<<=` before `<<` before `<`, etc. The token
  package already enumerates the multi-char operator kinds.
- Col is a 1-based rune/byte column reset on `\n`; Offset is the 0-based byte
  index of the token start; Line is 1-based.
- Keep the package import-light and stdlib-only (project is zero-dependency).
- Skip whitespace between tokens; comments (`//`, `/* */`) are recognized so
  later code can attach them, but goal `///` doc-comment trivia is US-013.

## Out of scope guard

Do NOT emit QUESTION / FAT_ARROW / ELLIPSIS here — those single-token goal
lexemes are US-013. A core-Go sample contains none of them, so deferring keeps
this story's surface minimal.
