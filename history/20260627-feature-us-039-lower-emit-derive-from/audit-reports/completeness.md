# Audit — Completeness

## Findings

No CRITICAL or MAJOR findings. The spec is anchored to three concrete, checked-in
example cases (slice / from_storage / to_storage) and a known-good reference
lowering, so the behavior space is closed.

- MINOR: The spec describes container recursion generally (FR-4) but the corpus
  only exercises slice recursion and a direct pointer-leaf conversion. Array/map
  recursion are ported for completeness but are not corpus-gated. Acceptable —
  they mirror the proven splice encoding.
- MINOR: "same-named source field" matching is case-insensitive in the reference
  implementation; the spec does not state the case sensitivity. Non-blocking — the
  corpus field names match exactly.

## Assumptions

- Field name matching is case-insensitive (mirrors the legacy deriver).
- Generated temporary names are scope-aware gensyms; exact golden parity is
  deferred to US-042.
- Only the three file-mode features/12 cases are gated; the foreign package fixture
  remains on the splice engine (US-009).
