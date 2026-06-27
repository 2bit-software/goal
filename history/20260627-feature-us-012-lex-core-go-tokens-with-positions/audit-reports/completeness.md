# Audit: Completeness — US-012

## Findings

### MINOR — Column unit unspecified
FR-2 says "1-based Column" but does not state byte vs rune column. For the
ASCII-only corpus this is immaterial. Decision: byte column (offset minus
line-start + 1). Recorded as an assumption; not blocking.

### MINOR — Multi-char operator longest-match not called out
The spec lists "operators" generically. The token vocabulary includes
multi-char operators (`<<=`, `&^`, `:=`, ...). The lexer must longest-match
these. This is implied by "tokenizes ... operators" and the token package; no
spec change needed.

### MINOR — Comment token shape
FR-4 says comments "SHALL be recognized so later stages can consume them" but
does not pin whether they are emitted in the stream or skipped. Either satisfies
this story (the US-012 test sample is comment-free). Decision: emit COMMENT
tokens (so US-013 can reclassify `///`). Recorded as an assumption.

No CRITICAL or MAJOR findings. The functional requirements cover the happy path
(all lexeme classes), positions, keyword/ident split, EOF, and the ILLEGAL
error path. Out-of-scope is explicit and matches the PRD's US-013 boundary.

## Assumptions

- Column is a 1-based byte column, reset at each newline.
- Comments are emitted as COMMENT tokens (not silently dropped).
- Source is treated as UTF-8; identifier letters use Unicode letter/digit
  classification (matching Go), though the corpus is ASCII.
- `\r\n` and `\r` are tolerated; only `\n` increments the line counter
  (Go-compatible: a bare `\r` is whitespace).
