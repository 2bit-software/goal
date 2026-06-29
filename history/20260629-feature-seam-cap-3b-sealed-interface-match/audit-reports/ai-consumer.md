# AI-Consumer Readiness Audit — SEAM-CAP-3b

No CRITICAL or MAJOR findings. The technical-requirements-research.md names exact
files, functions, and line numbers for all four gaps, plus the mirror set. An
implementer can proceed without guessing.

## MINOR

- MINOR: The exact Go-render of a `case *T:` label relies on the emitter's existing
  expr renderer for StarExpr/Ident/SelectorExpr; confirm `e.expr` handles a bare
  type expression in case-label position (it does — same nodes used elsewhere).
- MINOR: Registry field naming (`SealedImpls`) and the decision to keep `Sealed`
  bool are documented with rationale (avoids feature-07 regression). An implementer
  should mirror the field into selfhost/sema/sema.goal's Info struct identically.

## Assumptions

- `SealedImpls map[string][]string` holds implements-relations for ALL interfaces
  (sealed or not); it is only consulted for keys that are also sealed, so non-sealed
  entries are inert. This sidesteps cross-file ordering (sealed decl vs implementor
  struct in different files) without a second reconciliation pass.
- Union-with-dedup in Info.Merge is the correct multi-file same-package semantics
  for the registry (a sealed interface's implementors may be spread across files).
