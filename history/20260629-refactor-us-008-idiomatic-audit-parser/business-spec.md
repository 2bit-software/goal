# US-008 Idiomatic audit: parser — Business Specification

## Overview
Step 3 (idiomatic audit) of the self-host idiomatic plan, applied to
`selfhost/parser`. The goal is that the self-hosted parser reads as idiomatic
goal: parse error handling uses the goal `Result`/`?` idiom and any in-file
`switch`-over-`enum` uses `match` — wherever an intra-package, behavior-preserving
conversion exists. Every place a Go-ism is deliberately kept is recorded with a
reason in DECISIONS.md. The US-003 verbatim self-host is the behavioral oracle and
must stay byte-identical.

## Functional Requirements

### FR-1: Idiomatic error handling where it fits
Any package-internal parser helper that returns `(T, error)` and propagates with a
manual `if err != nil` SHALL be expressed with `Result` + `?` propagation, unless
doing so would break a public signature the oracle pins or require cross-package
caller edits — in which case the decision is recorded in DECISIONS.md.

### FR-2: match where an in-file enum is switched
Any `switch` over an in-file goal `enum` SHALL be expressed as `match`. Switches
over non-enum scrutinees (`token.Kind` int, tokens, `ast` interface type-switches,
boolean conditions) remain plain `switch`.

### FR-3: Public API preserved
The exported entry point `ParseFile(src string) (*ast.File, error)` SHALL keep its
signature unchanged.

## Acceptance Criteria

- [ ] Parser functions returning `(node, error)` use `Result` with `?`
      propagation where an intra-package behavior-preserving conversion exists;
      otherwise the refusal is recorded in DECISIONS.md with a specific reason.
- [ ] In-file `switch`-over-`enum` becomes `match` where it fits; non-enum
      switches are left as-is (recorded).
- [ ] `goal fix` reports no remaining auto-convertible propagation sites in
      `selfhost/parser`.
- [ ] parser tests pass against the transpiled package (`internal/selfhost` port
      gate under `task check`).
- [ ] `task check`, `task build`, and `task fixpoint` are green, with the fixpoint
      byte-identical.

## User Interactions
None directly. The artifact is the self-hosted compiler's parser source plus a
DECISIONS.md ledger entry.

## Error Handling
Behavior is unchanged: the parser keeps its error-accumulator design (errors
collected in `parser.errs`, joined by `ParseFile`). No user-visible error behavior
changes.

## Out of Scope
- Cross-package conversions (sealing `ast` interfaces, converting `token.Kind`),
  which belong to the whole-tree US-013 sweep.
- Any change to the public `ParseFile` signature.
- Changes to packages other than `selfhost/parser`.

## Open Questions
None. The audit outcome is determined by the source: the parser uses an
error-accumulator (no intra-package `(T,error)` propagation surface) and declares
no in-file `enum`, so the conversions named by the AC do not apply and are recorded
as genuine refusals-with-reason.
