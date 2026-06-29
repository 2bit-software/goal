# Technical requirements / research

## Chosen design: embedding cascade (option b)

Mirrors Go's own interface embedding; no parser change (rejects the multi-clause
`implements B, A` alternative, which would require new grammar).

### sema (resolve.go / resolve.goal)
- Add `cascadeSealedImpls()`: for each sealed iface B with implementors,
  propagate those implementors to every sealed interface A that B transitively
  embeds (walk `info.EmbeddedIfaces`, gate on `info.Sealed[A]`).
- `info.EmbeddedIfaces` is ALREADY populated for sealed interfaces:
  `resolveInterfaceMethods` is shared by `sealed interface` and `interface`, and
  records bare-type embeds. So `sealed interface B { A }` yields
  EmbeddedIfaces["B"]=["A"].
- Call cascade at the end of `Resolve(f)` (single-file path used by
  backend.Transpile) AND at the end of `ResolvePackage` after the merge loop
  (multi-file). Idempotent (addImplementor dedups).
- Cross-package propagation is free: goalForeignDecls projects the already-
  cascaded SealedImpls + Sealed; no foreign.go change needed.

### backend (emit.go / lower.go + .goal mirrors)
- `implementsMarker`: when iface is sealed, emit its marker AND a marker for each
  transitively-embedded sealed interface (new `sealedEmbeds(info, iface)` helper
  in lower.go).

### Fixpoint safety
- selfhost source does not YET use nested sealed hierarchies, so emitted Go is
  unchanged for existing flat cases (cascade adds nothing; sealedEmbeds empty).

## Proof
- Regression fixture modeled on internal/backend/sealed_match_test.go: same
  package, sealed A, sealed B embeds A, concrete T implements B, match over A and
  over B; transpile + go build/test against reference type-switch.
- sema test: exhaustiveness enforced at both A and B levels.
