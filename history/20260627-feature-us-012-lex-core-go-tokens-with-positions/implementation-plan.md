# Implementation Plan — US-012 Lex core Go tokens with positions

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/lexer/lexer.go` | The lexer: a cursor over source that emits `[]token.Token` for the Go subset with accurate `token.Pos{Offset, Line, Col}`. |
| `internal/lexer/lexer_test.go` | Multi-line sample test asserting Kind + Pos of every token, plus focused tests for literals, keyword/ident split, longest-match operators, EOF, and ILLEGAL. |

### Modified Files
None. The token vocabulary (`internal/token`) is complete from US-011 and needs
no change.

## Package Structure

```
internal/
  token/        # US-011: Kind, Pos, Token, Lookup, IsKeyword (unchanged)
  lexer/        # NEW (this story)
    lexer.go
    lexer_test.go
```

`internal/lexer` imports only `internal/token` and stdlib (`unicode`,
`unicode/utf8`). It sits one layer above token in the front-end graph and below
the future `internal/parser`.

## Dependency Graph

1. `internal/token` (already exists) — Kind / Pos / Token / Lookup.
2. `internal/lexer/lexer.go` — depends on (1).
3. `internal/lexer/lexer_test.go` — depends on (1) and (2).

## Interface Contracts

```go
package lexer

// Lexer scans source into tokens, tracking byte offset, line, and column.
type Lexer struct { /* src, offset, rdOffset, ch, line, lineStart */ }

// New returns a Lexer positioned at the start of src.
func New(src string) *Lexer

// Next returns the next token; the final token has Kind == token.EOF and Next
// keeps returning EOF thereafter.
func (l *Lexer) Next() token.Token

// Tokens scans src to completion and returns all tokens including the trailing
// EOF. Convenience wrapper used by the test and future callers.
func Tokens(src string) []token.Token
```

Position semantics:
- `Offset` = 0-based byte index of the token's first byte.
- `Line` = 1-based, incremented after each `\n`.
- `Col` = 1-based byte column = `offset - lineStart + 1`.

Lexeme coverage (Go subset):
- Identifiers / keywords: `unicode.IsLetter | '_'` start, then letters/digits.
  Resolve via `token.Lookup` → keyword Kind, else `token.IDENT`.
- Numbers: INT / FLOAT / IMAG (decimal incl. `0x`,`0o`,`0b` prefixes, `.`,
  exponent, trailing `i`). One scan path classifying by what it consumes.
- Strings: `"..."` (STRING), raw `` `...` `` (STRING), runes `'...'` (CHAR),
  with escape handling.
- Operators / delimiters: `switch ch` with nested longest-match for multi-char
  forms (`<`,`<=`,`<<`,`<<=`,`<-`; `&`,`&=`,`&&`,`&^`,`&^=`; etc.), mapped to
  the operator Kinds in `internal/token`.
- Comments: `//` line and `/* */` block → `token.COMMENT` (Lit = comment text).
- Whitespace (` `, `\t`, `\r`, `\n`) skipped; `\n` advances line.
- Unknown byte → `token.ILLEGAL` token (Lit = the byte) so the parser reports it.

Out of scope (US-013): `?`, `=>`, `...`, `///` doc comments, trivia attachment.

## Integration Points

No existing source is modified. The lexer is consumed later by
`internal/parser` (US-017+). For now its only consumer is its own test. The
legacy `internal/scan.Lex` (text/scanner based) is untouched and remains in use
by the splice passes during Phase 1.

## Testing Strategy

`internal/lexer/lexer_test.go`, package `lexer` (internal test; it only needs
`token` + `lexer`). Stdlib `testing` only — no testify (project constraint).

- `TestTokenizeMultiLineSample` (the acceptance test): a fixed multi-line goal/Go
  source string; an expected `[]token.Token` (Kind, Lit where relevant, and full
  Pos) computed by hand; assert element-by-element equality and that the last
  token is EOF at the right position. Exercises line increment + col reset.
- `TestLiterals`: INT/FLOAT/IMAG/CHAR/STRING/raw-string each classify correctly.
- `TestKeywordVsIdent`: `func`/`for`/`return` → keyword Kinds; `implements`,
  `sealed`, `from`, `derive`, `foo` → IDENT.
- `TestLongestMatchOperators`: `:=`, `<<=`, `&^`, `==`, `...`-free set produce
  single multi-char tokens.
- `TestIllegal`: an unexpected byte yields an ILLEGAL token, no panic.
