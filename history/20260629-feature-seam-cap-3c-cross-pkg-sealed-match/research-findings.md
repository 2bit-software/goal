# Research Findings

This is a codebase-internal capability extension; no external/library research needed.
The relevant prior art is in-repo (SEAM-CAP-2 enum projection, SEAM-CAP-3b same-package
sealed match). Verified facts from reading the source:

## Summary
- `sema.EnrichForeign` merges structs/funcs/methods/enums into `info`; it does not touch
  `info.Sealed` / `info.SealedImpls`. (internal/sema/foreign.go:95-112, mirror in selfhost.)
- `goalForeignDecls` (the SEAM-CAP-2 .goal-source path) projects EXPORTED enums only.
- `ResolvePackage` already builds `info.Sealed` (bare iface names) and
  `info.SealedImpls` (bare iface → `*T` implementor list) from `sealed interface` +
  `implements` clauses (resolve.go:81, 101-109).
- Backend `sealedMatch` lowering is dispatched by `isSealedMatch(m)` (pattern-shape only;
  any TypePattern arm) and renders `case *pkg.T:` directly from the pattern type — it does
  NOT consult the registry, so cross-package match ALREADY lowers correctly. (emit.go:2159+)
- The sema gap: `checkOneSealedMatch` → `sealedInterfaceOf` finds nothing cross-package and
  emits `unresolved-match-sealed` Warning (check.go:213-223) — the exact boundary CAP-3b
  left for CAP-3c.
- `qualifyForeignType("*Lit", "shape")` → `*shape.Lit`, matching `typeString` of a
  `*shape.Lit` type pattern (resolve.go:461-464) — so registry keys align by string.

## Confidence
High — all facts read directly from the current tree; single call site for foreignDecls.

## Approach chosen
Extend the goal-source projection path (smallest change, matches SEAM-CAP-2 precedent and
the AC). Add a 6th `sealed map[string][]string` return to foreignDecls/goalForeignDecls;
merge into info.Sealed + info.SealedImpls in EnrichForeign. .go path returns nil sealed.

## Open questions
None blocking. .go-path sealed reconstruction is deferred (real build uses .goal siblings),
consistent with SEAM-CAP-2 deferring struct/func/method facts from .goal.
