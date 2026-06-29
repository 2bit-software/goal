# SEAM-006 Business Spec

## Outcome

The self-hosted goal compiler demonstrably reads as idiomatic goal rather than
transpiled Go, and that claim is *measured*, not asserted.

## Acceptance criteria

1. `task fixpoint` green (goal-c-1 and goal-c-2 byte-identical on the new
   idiomatic source) AND the full corpus behavioral + interp + check tiers green.
2. A DECISIONS.md writeup tallies, per seam, what converted and what remains a
   documented semantic non-fit, and states the honest end state (which idioms
   the compiler now showcases tree-wide: enum, match over sealed AST, Result/?).
3. `goal fix` over the whole selfhost tree reports no remaining auto-convertible
   propagation (the autofixer agrees the propagating API is idiomatic). Residual
   suggestions, if any, recorded honestly.
4. A short before/after in DECISIONS.md or SELF-HOST-RESEARCH.md quantifies the
   shift (count of type-switches now match, iota types now enum, fallible APIs
   now Result) so the result is measurable.

## Constraints

- Documentation + proof only; no further source idiom changes beyond what the
  earlier seams landed.
- Numbers must be grepped/counted from the actual tree, never invented.
