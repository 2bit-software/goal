# Lex core Go tokens with positions — Business Specification

## Overview

The goal AST front-end needs a lexer: the stage that converts raw source text
into a flat stream of tokens. This story delivers that lexer for the *Go subset*
goal is built on — ordinary identifiers, keywords, literals, operators, and
delimiters — with every token carrying an accurate source position. Accurate
positions are the whole point: the legacy splice passes re-lexed and discarded
byte offsets every pass, so diagnostics could never point at exact source. The
lexer feeds the parser (US-017+) a reliable, position-bearing token stream.

## Functional Requirements

### FR-1: Tokenize the Go subset
The lexer SHALL produce a token for each lexeme of the Go subset: identifiers,
reserved keywords, integer / floating-point / imaginary / character / string
literals, and the operator and delimiter set defined in `internal/token`.

### FR-2: Accurate positions
Every emitted token SHALL carry a position with a 0-based byte Offset, a 1-based
Line, and a 1-based Column, all measured at the token's first character. Line
SHALL increment after each newline and Column SHALL reset to 1 at the start of
each line.

### FR-3: Keyword vs identifier
A word matching a goal reserved word SHALL be emitted as that keyword Kind;
any other word SHALL be emitted as IDENT. (The contextual keywords
implements/sealed/from/derive SHALL remain IDENT.)

### FR-4: Whitespace and EOF
The lexer SHALL skip inter-token whitespace and SHALL emit a terminating EOF
token positioned at end of input. Comments SHALL be recognized so later stages
can consume them.

## Acceptance Criteria

- [ ] The lexer tokenizes identifiers, literals, operators, and delimiters with
      accurate Offset/Line/Col.
- [ ] A test tokenizes a multi-line sample and asserts the Kind and position of
      each token.
- [ ] Keywords are distinguished from identifiers; contextual keywords stay IDENT.
- [ ] The token stream ends with an EOF token.
- [ ] `go build ./...`, `go vet ./...`, and `go test ./... -count=1` are green.

## User Interactions

Programmatic only: a Go API that accepts source text and returns the token
slice (and/or yields tokens one at a time). No CLI or UI surface in this story.

## Error Handling

Unrecognized input (a byte that begins no valid lexeme) SHALL be surfaced as an
ILLEGAL token carrying the offending text and position, rather than panicking,
so the parser can report it positionally.

## Out of Scope

- Goal-specific single-token lexemes `?` (QUESTION), `=>` (FAT_ARROW), and
  `...` (ELLIPSIS) — deferred to US-013.
- `///` doc comments (DOC_COMMENT) and attaching comments as trivia — US-013.
- Parsing / AST construction — later stories.

## Open Questions

None. Scope and the token vocabulary are fixed by US-011 and the PRD.
