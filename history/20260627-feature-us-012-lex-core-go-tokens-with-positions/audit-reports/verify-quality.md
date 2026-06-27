# Verify: Quality — US-012

## Checks
- **Error handling**: unrecognized bytes produce ILLEGAL tokens (with the
  offending text and position) and always advance the cursor, so the lexer can
  never spin or panic. Unterminated strings/runes terminate at newline/EOF
  rather than reading off the end (bounds guarded by `ch != eof`).
- **Edge cases**: blank lines and leading tabs are exercised by the multi-line
  acceptance test; hex/octal/binary/underscore/exponent/imaginary number forms
  are covered by `TestLiterals`; longest-match operator boundaries (`<`, `<<`,
  `<<=`, `&`, `&^`, `&^=`) by `TestLongestMatchOperators`.
- **Spec fidelity**: the deferred goal lexemes are deliberately NOT emitted — a
  `?` lexes as ILLEGAL, `=>` as ASSIGN+GTR, `.` as PERIOD — matching the
  US-013 boundary stated in the spec's Out of Scope. No code claims to handle
  them.
- **Tests assert real behavior**: each test compares concrete Kind/Lit/Pos
  values, not just non-empty output; the multi-line test pins exact offsets.

## Findings
No CRITICAL or MAJOR findings. One MINOR:

### MINOR — Number scanner is permissive
`scanNumber` accepts some forms the Go spec would reject (e.g. a leading-zero
decimal, or `_` adjacency edge cases). This is acceptable for a lexer whose job
is tokenization, not literal validation; the parser/go-backend and the
behavioral tier reject malformed numbers downstream. No corpus input depends on
strict literal validation at the lexer layer.

## Assumptions
- The lexer targets file-sized in-memory strings (no streaming); single-pass,
  non-concurrent.
- UTF-8 source; multi-byte runes advance the byte cursor via
  `utf8.DecodeRuneInString`, keeping Offset byte-accurate.
