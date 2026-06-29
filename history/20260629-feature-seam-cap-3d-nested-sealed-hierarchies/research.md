# Research — nested sealed-interface hierarchies

Codebase-internal research (this is a compiler-internal capability; no external
sources apply). Findings verified by reading internal/ and selfhost/ source.

## Existing machinery (reused)
- `sema.Info.EmbeddedIfaces map[string][]string` already records embedded
  interface names for BOTH ordinary and sealed interfaces, because
  `resolveInterfaceMethods` (resolve.go) is shared by `interface` and
  `sealed interface` and appends bare-type members to EmbeddedIfaces. So
  `sealed interface B { A }` → EmbeddedIfaces["B"]=["A"]. No new tracking needed.
- `sema.Info.SealedImpls map[string][]string` (CAP-3b) maps iface → `*T`
  implementors; `addImplementor` dedups. `Sealed map[string]bool` marks sealed.
- Backend `implementsMarker` (emit.go) emits `genMarkerMethod(T, iface)` =
  `func (T) isIface() {}` for a sealed iface.
- Parser already parses embedded interfaces inside a sealed interface body
  (parseMethodSpec → Field with no Names). So `sealed interface B { A }` parses.

## Design decision: embedding cascade (option b), no parser change
Option (a) multi-clause `implements B, A` requires new grammar in the parser and
duplicates Go's own embedding semantics by hand. Option (b) cascade mirrors Go
interface embedding and reuses EmbeddedIfaces — strictly less code, no grammar
change. Chosen.

Cascade = propagate B's implementors to every sealed interface B transitively
embeds (sema), and emit a marker for each (backend).

## Cross-package
goalForeignDecls (foreign.go, CAP-3c) projects info.Sealed + SealedImpls from a
sibling .goal package's ResolvePackage. Since the cascade runs inside
ResolvePackage, the projected sets are already cascaded → cross-package works
with NO foreign.go change. SEAM-004's AST is one package anyway.

## Confidence: High. Proof harness pattern: internal/backend/sealed_match_test.go.
