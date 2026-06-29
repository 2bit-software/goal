# Scaffold notes — US-006 idiomatic audit: lexer

## Outcome: documented decision, no .goal source change
The idiomatic audit of `selfhost/lexer/lexer.goal` finds the existing source is
already the correct idiomatic form. The "new implementation" is therefore a
recorded DECISION (refusals-with-reason), not a code rewrite — identical in
shape to US-005 (token).

## Files created/changed
- `DECISIONS.md` — appended section "self-host idiomatic audit — US-006 (lexer)"
  with two entries:
  1. Refusal: the lexer's `switch` statements stay `switch` (none are over an
     in-file enum; they switch over `rune` and boolean conditions, which
     DECISIONS §02-match §228 keeps as plain `switch` — `match` is for closed
     enums only).
  2. Assumption: no Result/Option/`?` conversion applies — the lexer is a total
     tokenizer with no fallible `(T,error)` helper; it signals lexical errors
     in-band via `token.ILLEGAL`. `goal fix` produces no diff/report.

## How the result differs from "old"
No behavioral or signature difference — that is the point. The audit confirms
the package already satisfies every AC:
- AC-1: switch→match / Result conversions — none fit (recorded refusals).
- AC-2: `goal fix selfhost/lexer/lexer.goal` → no diff, no report (verified).
- AC-3: lexer tests pass against the transpiled package; `task fixpoint` stays
  byte-identical (verified in the verify step).

## How to verify independently
- `./bin/goal fix selfhost/lexer/lexer.goal` → no stdout diff vs source, empty
  stderr.
- `task check` (includes the internal/selfhost port gate + internal/lexer
  tests), `task build`, `task fixpoint`.
