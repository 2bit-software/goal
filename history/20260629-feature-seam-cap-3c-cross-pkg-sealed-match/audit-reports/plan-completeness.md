# Plan Coverage Audit

## Coverage matrix
- AC "propagate sibling sealed implementor set (extend goal-source path)" → foreign.go
  goalForeignDecls projection. COVERED.
- AC "BOTH internal/ and selfhost/" → foreign.go + foreign.goal mirror. COVERED.
- AC "2+-package fixture, transpiles + behaves identically" → backend behavioral test +
  goalsealed fixtures. COVERED.
- AC "non-exhaustive Error / clean when complete" → sema crosspkg_sealed_test. COVERED.
- AC "gates green, corpus unchanged" → testing strategy names all three gates; fixtures
  additive. COVERED.

## Findings
- No CRITICAL/MAJOR. No scope creep: every plan element traces to an AC.
- MINOR: plan does not call out a DECISIONS.md entry; the SEAM PRD records each seam in
  DECISIONS.md — add one in the implement step for consistency with CAP-3a/3b.

## Assumptions
- Backend lowering needs no change (pattern-driven dispatch verified in research) — the
  whole capability is sema-side propagation.
