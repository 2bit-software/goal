# Idiomatic audit: token — Business Specification

## Overview

selfhost/token is the first package in the idiomatic-audit phase (step 3 of the
self-host idiomatic plan). The goal is to make selfhost/token read as idiomatic
goal rather than transpiled Go, while remaining behavior-preserving against the
verbatim self-host oracle (US-003). Token kinds should be expressed the way the
language intends — as a goal `enum` if that representation fits, otherwise the
deliberate decision to keep the iota const block must be recorded.

## Functional Requirements

### FR-1: Token-kind representation is idiomatic or justified
The token-kind set SHALL be expressed as a goal `enum` where that representation
fits the way the kinds are used; otherwise the deliberate decision to keep the
iota-based const block SHALL be recorded in DECISIONS.md with its rationale.

### FR-2: No auto-convertible error propagation remains
The package SHALL contain no manual `if err != nil { return ..., err }`
propagation that `goal fix` would convert to Result/`?`.

### FR-3: Behavior preserved
The package's observable behavior SHALL be unchanged: the existing token tests
pass against the transpiled package, and the byte-identical compiler fixpoint
holds.

## Acceptance Criteria

- [ ] iota-based token-kind const blocks are expressed as a goal `enum` where it
      fits, or the deliberate decision not to is recorded in DECISIONS.md.
- [ ] `goal fix` reports no remaining auto-convertible propagation sites for the
      token package.
- [ ] token tests pass against the transpiled package.
- [ ] `task fixpoint` stays green (goal-c-1 and goal-c-2 byte-identical).
- [ ] `task check` and `task build` are green.

## User Interactions

None directly. This is internal compiler source. The audit is observed through
the verification gates (`task check`, `task build`, `task fixpoint`) and the
DECISIONS.md ledger.

## Error Handling

No new error surfaces. The token package is import-free and has no fallible
operations; `Lookup` keeps its comma-ok `(Kind, bool)` contract.

## Out of Scope

- Any change to the token package's public API shape (the oracle tests are
  reused unchanged against the transpiled package).
- Idiomatic audits of other selfhost packages (lexer, ast, parser, ... — their
  own stories US-006+).
- Converting `Lookup` to `Option[Kind]` (would break the reused oracle test and
  is not a goal-fix propagation site).

## Open Questions

None. The audit findings (technical-requirements-research.md) are conclusive:
the goal `enum` (sealed-interface encoding) does not fit an array-indexed,
range-marker integer Kind, so the decision is to keep the iota const block and
record the rationale in DECISIONS.md.
