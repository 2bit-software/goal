# Scaffold notes — US-043

## The replacement already exists

This refactor removes the splice front-end. Its replacement — the AST backend
(`internal/backend.Transpile` / `backend.TranspilePackage`) — was built and
proven across US-026..US-042 and has been the default engine since US-042. No
new production code needs to be written; the work is:

1. Re-point the remaining splice-engine call sites onto `backend.*`.
2. Delete `internal/pass`, the `internal/pipeline` splice functions, and the
   `scan`/`analyze` symbols that become dead.

Both are deletion/cutover actions, performed in the cutover step (the scaffold
step forbids modifying existing code).

## Independent testability of the replacement

The AST backend is already exercised by:
- `internal/corpus.TestASTEngineWholeCorpusBehavioralGate` (all 108 manifest
  cases through the AST engine at the behavioral tier).
- The exact-tier corpus tests (`TestTranspileRunner`, `TestDoctestRunner`,
  `TestCorpusTranspile`, `TestCorpusDoctest`) against goldens regenerated from
  the backend.
- `internal/backend/*_test.go` per-construct lowering tests.

So "verify the new code works" is already covered by the green suite; the
cutover only has to keep it green after the splice removal.

## Coexistence

No adapters needed — `backend.Transpile` and `pipeline.Transpile` share the same
`pipeline.Output` shape, and `backend.TranspilePackage` already returns
`pipeline.PackageOutput`. Call sites swap one function for another of identical
signature.
