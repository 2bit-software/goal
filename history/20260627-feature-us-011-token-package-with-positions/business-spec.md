# Token Package with Positions — Business Specification

## Overview

The goal compiler is moving from a token-splice front-end to a real
lexer → parser → AST pipeline. This feature lays the first foundation stone: a
shared vocabulary of token kinds for every goal lexeme and a source-position type
that records where each lexeme appears. Today every pass re-lexes the source and
throws byte offsets away; with first-class positions the future AST can carry real
line/column locations for diagnostics, fmt, and LSP.

## Functional Requirements

### FR-1: Token kinds for every goal lexeme
The system SHALL define a named `Kind` value for every lexeme of the goal language:
ordinary Go tokens (identifiers, the literal classes, every operator and delimiter,
and the reserved keywords) plus the goal-specific lexemes that the splice approach
faked — postfix `?`, the fat arrow `=>` (one token, not `=` then `>`), the ellipsis
`...`, and `///` doc-comment trivia. The set SHALL also include the structural
sentinels `ILLEGAL` (unrecognized input) and `EOF` (end of input).

### FR-2: Contextual keywords are identifiers
The system SHALL lex `implements`, `sealed`, `from`, and `derive` as ordinary
identifiers, not as distinct kinds; the parser decides them positionally. Only true
reserved words receive keyword kinds.

### FR-3: Stable, round-trippable kind names
Each `Kind` SHALL have a stable, human-readable string name via a `String()` method.
For every non-sentinel keyword/lexeme kind, looking that name back up SHALL return
the identical `Kind` (name → Kind → name round-trips).

### FR-4: Source positions
The system SHALL define `Pos{Offset, Line, Col}` recording a 0-based byte offset and
1-based line/column. Positions SHALL be comparable so that a position earlier in the
source orders strictly before one later in the source, using `Offset` as the
canonical total order.

## Acceptance Criteria

- [ ] `internal/token` defines a `Kind` named type with constants covering every
      goal lexeme (Go tokens + `?`, `=>`, `...`, `///`, keywords, `ILLEGAL`, `EOF`).
- [ ] `implements`/`sealed`/`from`/`derive` have NO dedicated kind (they are `IDENT`).
- [ ] `Kind.String()` returns a stable name and `Lookup(name)` round-trips back to the
      same `Kind` for every keyword/punctuation lexeme.
- [ ] `Pos{Offset, Line, Col}` exists with a comparison such that an earlier offset is
      ordered before a later one.
- [ ] A stdlib-only unit test asserts the `Kind`/`String` round-trip and `Pos`
      ordering, and it passes under `go test ./... -count=1`.

## User Interactions

This is an internal compiler library (`internal/token`). Consumers are the lexer
(US-012/013), AST (US-014+), and tools. There is no CLI or user-facing surface.

## Error Handling

Unrecognized input is represented by the `ILLEGAL` kind rather than a panic.
`Lookup` of an unknown name returns a not-found signal (the `ILLEGAL` kind / false),
never a panic.

## Out of Scope

- The lexer itself (US-012/013) — this story only defines the kinds and positions.
- The AST node types (US-014+).
- Trivia attachment / comment grouping machinery beyond defining the `///` kind.
- Any change to the existing splice front-end (`internal/scan`, passes).

## Open Questions

- None. The lexeme inventory is fixed by REWRITE-ARCHITECTURE.md §1.1/§1.2 and the
  existing `internal/scan` keyword set.
