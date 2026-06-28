# Plan Audit: Buildability — US-012

## Checks

- **Dependency order valid**: token (exists) → lexer.go → lexer_test.go. No
  forward references; lexer imports only `internal/token` + stdlib. Compiles at
  each step.
- **Interface contracts concrete**: `New(src string) *Lexer`,
  `(*Lexer) Next() token.Token`, `Tokens(src string) []token.Token` are full Go
  signatures using the real `token.Token` type from US-011.
- **File paths verified**: `internal/lexer/` does not yet exist (confirmed —
  `ls internal/` shows no lexer dir), so no path conflict. `internal/token`
  exists with the referenced `Kind`, `Pos`, `Token`, `Lookup`, `IsKeyword`.
- **Integration points specific**: no existing file is modified; the only
  consumer this story is the test. Legacy `internal/scan.Lex` is explicitly left
  untouched (separate text/scanner path used by splice passes).
- **Position arithmetic specified**: Col = offset - lineStart + 1; line
  increments after `\n`. Unambiguous to implement.

## Findings

No CRITICAL or MAJOR findings. One MINOR:

### MINOR — `Next()` vs slice API
Both `Next()` (incremental) and `Tokens()` (whole-slice) are specified. This is
intentional (parser will want incremental; the test wants the slice). Minor
surface duplication, not a blocker; `Tokens` is a thin loop over `Next`.

## Assumptions

- Lexer is non-concurrent / single-pass over an in-memory string (the project
  transpiles file-sized inputs; no streaming requirement).
- UTF-8 source; `unicode/utf8.DecodeRuneInString` advances the cursor for
  multi-byte runes so Offset stays byte-accurate.
