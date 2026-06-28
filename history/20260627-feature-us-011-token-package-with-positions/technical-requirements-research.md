# Technical Requirements & Research — US-011

## Source of truth

- REWRITE-ARCHITECTURE.md §1.1 (`token`): "Token kinds + `Pos{Offset, Line, Col}`.
  Positions are first-class (today's offsets are thrown away every pass)."
- REWRITE-ARCHITECTURE.md §1.2 enumerates the goal-specific lexemes the lexer must
  emit: `?` as a token, `=>` as ONE token (not `=` then `>`), `...` as one token,
  `///` doc-comment content retained as trivia, and `implements`/`sealed`/`from`/
  `derive` lexed as identifiers (contextual keywords).

## Goal lexeme inventory (from internal/scan/scan.go)

Stmt-leading keywords already recognized by scan.IsStmtKeyword: return, go, defer,
if, else, for, switch, select, case, default, var, const, type, func, range, break,
continue, goto, fallthrough, match, assert, enum, import, package. Plus Go keywords
not in that list: struct, interface, map, chan, nil, true, false. Operators and
delimiters per Go, plus goal additions: `?` (UnwrapExpr), `=>` (match arm arrow),
`...` (ellipsis/spread), `///` (doc comment trivia), `:` label separator (already a
Go token).

## Design decisions

- `Kind` is an `int`-backed named type with `iota` constants and a `String()` method
  (the round-trip lookup uses a name->Kind map built from the same table).
- Contextual keywords stay `IDENT` — no distinct Kind. Only true reserved words get
  keyword Kinds (use Go's keyword set + goal's `match`/`enum`/`assert`).
- `Pos{Offset, Line, Col int}` with a `Less(other) bool` (or `Before`) comparison
  primarily ordered by Offset (Offset is the canonical total order; Line/Col are
  derived but carried for diagnostics).
- A `Token{Kind, Lit string, Pos}` type is reasonable to include as the unit the
  lexer will emit, but US-011's gate is only Kind + Pos; keep Token minimal.

## No new dependencies. Stdlib only.
