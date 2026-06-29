# Verify: Acceptance Coverage

Full gates run green: `task check` (go vet + full test suite, includes
internal/corpus behavioral tier), `task build`, `task fixpoint` (FIXPOINT OK).

| Acceptance criterion | Evidence |
|---|---|
| Sealed interface with methods keeps signatures + marker | `TestSealedInterfacePreservesMethodSignatures` asserts `Pos() Position`, `End() Position`, `isNode()` present; also asserts the compact marker-only form is NOT used. |
| Empty-body sealed interface unchanged | `TestSealedInterfaceEmptyBodyStaysCompact` asserts `type Shape interface{ isShape() }`. |
| Implementors compile + methods callable through interface | `TestSealedInterfaceMethodsCallableThroughInterface` builds + `go test`s a temp module where a `Leaf` implementor is assigned to a `Node` value and `Pos()`/`End()`/`startLine` are called through it. |
| Fixed in internal/ AND selfhost/ | emit.go + emit.goal both carry the identical `interfaceMethod` helper + `sealedInterfaceDecl` rewrite; selfhost mirror verified by `task fixpoint` and internal/selfhost port tests. |
| task check/build/fixpoint green; corpus unchanged | All three gates green; internal/corpus passed under task check. |

No acceptance criterion is uncovered.

## Assumptions
- Callability is proven via a temp-module `go test` (existing backend pattern).
- Empty-body sealed interfaces deliberately retain the compact form to keep
  emitted Go byte-identical (fixpoint protection).
