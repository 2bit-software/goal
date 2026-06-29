# Technical Requirements / Research

## Root-cause (verified)

- `sema.EnrichForeign` (internal/sema/foreign.go + selfhost/sema/foreign.goal) merges
  only structs/funcs/methods/enums into `info`. It never touches `info.Sealed` or
  `info.SealedImpls`.
- `foreignDecls` has two source paths (SEAM-CAP-2): a `.go` reconstruction path and the
  `.goal`-source `goalForeignDecls` path. `goalForeignDecls` runs
  `parser.ParseFile + ResolvePackage` and projects EXPORTED enums only.
- Consequence: a consumer's `checkOneSealedMatch` calls `sealedInterfaceOf(info, "*pkg.T")`,
  finds nothing in `info.SealedImpls`, and emits the `unresolved-match-sealed` Warning
  (the boundary CAP-3b explicitly left for CAP-3c).
- Backend `sealedMatch` lowering (emit.go) is dispatched by `isSealedMatch(m)` which is
  purely pattern-shape driven (any TypePattern arm) — it does NOT consult the registry,
  so cross-package match ALREADY lowers to `case *pkg.T:`. The only gap is sema resolution.

## Fix

Extend the goal-source path to also project sealed interfaces:

1. `foreignDecls` / `goalForeignDecls` gain a 6th return value
   `sealed map[string][]string` (sealed iface qualified name `alias.Iface` →
   qualified implementor list `*alias.T`).
2. `goalForeignDecls`: after `ResolvePackage`, for each EXPORTED iface in `info.Sealed`,
   project `sealed[alias+"."+iface] = qualifyForeignType(impl, alias)` for each impl in
   `info.SealedImpls[iface]` (qualifyForeignType already turns `*Lit` → `*alias.Lit`).
3. `.go` path returns `nil` for sealed (out of scope — the real build resolves siblings to
   .goal; reconstructing sealed from generated .go is deferred, like struct/func .goal facts).
4. `EnrichForeign`: nil-init `info.Sealed`/`info.SealedImpls`, then for each `(iface, impls)`
   set `info.Sealed[iface] = true` and `info.SealedImpls[iface] = impls`.

Mirror identically in internal/ (Go) and selfhost/ (.goal).

## Proof

- internal/sema cross-package sealed enrichment + exhaustiveness test (sibling-.goal fixture).
- internal/backend behavioral test (mirrors crosspkg_goal_enum_test.go): transpile the
  defining .goal package AND the consumer per-package, build into a temp `module goal`,
  run against a reference `switch x := n.(type)`.

## Gates

`task check`, `task build`, `task fixpoint` (watch closely — touches the transpile
pipeline); corpus behavioral tier unchanged (additive fixture).
