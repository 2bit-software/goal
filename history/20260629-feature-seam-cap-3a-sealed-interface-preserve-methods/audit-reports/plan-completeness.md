# Plan Audit: Coverage

## Findings

- FR-1 (preserve methods) -> `sealedInterfaceDecl` rewrite + `interfaceMethod`
  helper. Covered.
- FR-2 (marker retained) -> emit `isName()` after methods; empty case via
  `genSealedInterface`. Covered.
- FR-3 (empty unchanged) -> compact-form branch kept. Covered.
- FR-4 (implementors compile / callable) -> behavioral build test. Covered.
- FR-5 (live + selfhost agree) -> mirror edit in selfhost/backend/emit.goal +
  fixpoint gate. Covered.

No plan element lacks a requirement; no scope creep (genSealedInterface kept,
lower.go untouched). No CRITICAL/MAJOR.

## Assumptions

- Keeping `genSealedInterface` for the empty case rather than folding it into the
  emitter is a deliberate fixpoint-protecting choice.
