# Scope — US-006 idiomatic audit: lexer

## What is being audited and why
selfhost/lexer/lexer.goal — the goal tokenizer. Step 3 of the self-host
idiomatic plan: make the package read as idiomatic goal (enum→match,
Result/Option + `?`) wherever the idiom fits, while staying behavior-identical
to the US-003 verbatim oracle.

## What the old (current) code looks like
- One file, `selfhost/lexer/lexer.goal`, package `lexer`.
- Imports `unicode`, `unicode/utf8`, `goal/selfhost/token`.
- `Lexer` struct + `New`, `next`, `peek`, `peek2`, `pos`, `Tokens`, `Next`,
  `skipWhitespace`, and the `scan*` family.
- Switches:
  - `Next()` — an expression-less switch over boolean conditions
    (`ch == eof`, `isLetter(ch)`, `isDigit(ch)`, `ch == '"'`, ...).
  - `scanOperator()` and its nested switches — switch over `l.ch` / `ch`,
    a `rune` (primitive integer), to pick the operator/delimiter token.
- No function returns `error`; the lexer reports lexical problems by emitting
  `token.ILLEGAL` tokens (scanOperator default), never by a fallible signature.
- `token.Lookup(lit)` returns comma-ok `(token.Kind, bool)`, called in
  `scanIdentifier`.

## What the new code should look like (goals, constraints)
- Convert switch→match ONLY where the switch is over an in-file closed `enum`.
- Convert fallible `(T, error)` helpers to Result/Option + `?` where natural.
- Record any refusal-with-reason in DECISIONS.md.

## What must NOT change (preserved interfaces)
- Public API the US-003 oracle pins: `New`, `Tokens`, `(*Lexer).Next`, and the
  token kinds/positions they emit — byte-for-byte behavior.
- `task fixpoint` must stay byte-identical (goal-c-1 vs goal-c-2).
- lexer tests must still pass against the transpiled package.

## Audit finding (drives the implementation step)
- No in-file `enum` exists in lexer. Per DECISIONS §02-match (§228), plain
  `switch` is legal on non-enum types and `match` is reserved for closed enums;
  the lexer's switches are over `rune` and boolean conditions, so none convert
  to `match`. (Mirrors US-005: `token.Kind` was kept as `type Kind int`, not an
  enum, so even token-kind switches would not be enum switches.)
- No fallible `(T, error)` helper exists — the lexer emits `token.ILLEGAL`
  rather than returning errors — so there is no Result/Option/`?` site.
- `token.Lookup` is comma-ok and pinned by the oracle; not a goal-fix site
  (US-005 precedent), left unchanged.
- Machine check already green: `goal fix selfhost/lexer/lexer.goal` produces no
  diff and no report.
- Outcome: a recorded DECISION (no .goal source change), exactly like US-005.
