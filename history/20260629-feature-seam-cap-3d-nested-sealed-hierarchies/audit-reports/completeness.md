# Completeness audit

## Findings

- MINOR: FR-5 (transitivity) is specified but the primary proof fixture is 2-level.
  Transitivity falls out of a recursive/transitive-closure walk; a 3-level case is
  worth a lightweight assertion but not blocking.
- MINOR: The spec relies on the existing `non-exhaustive-match` and
  `unresolved-match-sealed` diagnostics rather than introducing new ones — correct
  and intentional; no new diagnostic catalog entries expected.
- None CRITICAL or MAJOR. The spec maps directly onto existing CAP-3b machinery
  (SealedImpls, EmbeddedIfaces, implementsMarker) with a single new cascade.

## Assumptions

- The cascade is computed over the fully-merged package Info (run in both
  single-file Resolve and post-merge ResolvePackage) so cross-file hierarchies
  resolve. Idempotent via addImplementor dedup.
- Cross-package nested hierarchies need no foreign.go change because the cascade
  runs inside the foreign package's ResolvePackage before projection.
- Fixpoint stays byte-identical because selfhost source contains no nested sealed
  hierarchy yet (SEAM-004 introduces it); existing flat cases are untouched.
