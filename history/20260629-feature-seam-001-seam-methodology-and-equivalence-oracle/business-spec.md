# SEAM-001 Seam Methodology & Equivalence Oracle — Business Specification

## Overview

The per-package idiomatic audits (US-005..US-013) operated under a gate that
required emitted Go to stay byte-identical and oracle-pinned signatures to stay
fixed. That gate is exactly what every deep cross-package idiom (seal AST ->
match, iota -> enum, fallible API -> Result/?) must violate. The SEAM PRD relaxes
the gate. This story produces the written, repeatable definition of
"behavior-preserving" under the relaxed gate so that every later seam story has a
single, unambiguous oracle to verify against. It is documentation and procedure
only — no source idiom change.

## Functional Requirements

### FR-1: Seam methodology section in the decision ledger
DECISIONS.md SHALL contain a "Seam methodology" section that states the contrast
between the two gates: per-package audits required unchanged (byte-identical)
emitted Go; seam stories explicitly ALLOW emitted-Go change and re-prove
equivalence via (a) fixpoint self-consistency, (b) the corpus behavioral tier,
and (c) reviewed golden regeneration.

### FR-2: Test classification under a seam edit
The section SHALL enumerate which existing tests are EXPECTED to change under a
seam edit (the ported go/ast-mirror unit tests; golden transpile-shape fixtures)
versus which must stay byte-green (task fixpoint; corpus behavioral/interp tiers;
the full task check after golden regeneration).

### FR-3: Golden regeneration & review procedure
The section SHALL document a repeatable procedure for regenerating and reviewing
goldens: which target / update flag regenerates each kind of golden, and what a
reviewer checks so that a deliberate emitted-Go change cannot read as an
undetected regression.

## Acceptance Criteria

- [ ] DECISIONS.md has a "Seam methodology" section stating the relaxed gate and
      its three equivalence proofs (fixpoint self-consistency, corpus behavioral
      tier, reviewed golden regeneration), contrasted with the per-package gate.
- [ ] The section lists EXPECTED-to-change tests (go/ast-mirror units, golden
      transpile-shape fixtures) vs MUST-stay-green tests (task fixpoint, corpus
      behavioral/interp tiers, full task check after golden regen).
- [ ] The section documents the regenerate-and-review procedure (the corpus
      `-update-goldens` target, the parser `-update-snapshots` flag, and the
      reviewer's checklist).
- [ ] `task check`, `task build`, `task fixpoint` are all green (documentation +
      procedure only; no source idiom change).

## User Interactions

Audience is the engineers (and the loop) executing SEAM-002..006. They read the
section in DECISIONS.md to know what "green" means for a seam edit and how to
regenerate goldens. No runtime/CLI/UI surface changes.

## Error Handling

Not applicable — no executable behavior is added or changed.

## Out of Scope

- Any `.goal` / `.go` source idiom change (those are SEAM-002..006).
- Any golden regeneration in this story (no emitted Go changes here).
- Changes to Taskfile targets or test flags (the mechanisms already exist).

## Open Questions

- None. The fixpoint semantics and golden-update mechanisms already exist in the
  tree and are documented in the research artifact; this story only records the
  methodology that uses them.
