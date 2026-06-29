# SEAM-CAP-2 cross-.goal-package enum propagation — Business Specification

## Overview

When the goal compiler transpiles a multi-package goal project package-by-package (the
real `goal build ./selfhost` topology), an enum defined in one `.goal` package must be
usable from another `.goal` package. Today a cross-package enum only resolves when the
defining package already exists as generated Go; an enum defined purely in sibling `.goal`
SOURCE is invisible to its consumers, so `match` over it and bare construction of its
variants fail to transpile. This feature makes such an enum visible to consumers during
the per-package transpile and lowers cross-package variant construction correctly.

## Functional Requirements

### FR-1: Sibling-.goal enum visibility
During per-package transpile, when a consumer package imports a sibling package that
exists only as `.goal` source, the importer SHALL resolve that package's exported enums
(name set, variants, and field membership) so cross-package constructs over them lower.

### FR-2: Cross-package match lowers
A `match` whose arms name `pkg.Enum.Variant` over an enum defined in a sibling `.goal`
package SHALL transpile to a Go type-switch over the §8.1 sum encoding
(`case pkg.Enum_Variant:`) and SHALL behave identically to the equivalent hand-written
switch.

### FR-3: Cross-package construction lowers
Bare construction of a cross-package variant (`pkg.Enum.Variant`) SHALL lower to the
§8.1 form `pkg.Enum(pkg.Enum_Variant{})`, never verbatim.

### FR-4: Self-host consistency
The capability SHALL exist in both the live transpiler and the `.goal` self-host mirror,
so the self-host fixpoint (both bootstrap stages byte-identical) continues to hold.

## Acceptance Criteria

- [ ] A 2-package fixture where the enum is DEFINED IN A SIBLING `.goal` PACKAGE (not a
      `.go` fixture) transpiles a cross-package `match` without
      `unsupported expression *ast.MatchExpr` / `unsupported statement-position match`.
- [ ] The transpiled consumer Go contains the §8.1 type-switch over the imported variants.
- [ ] The transpiled consumer behaves identically to the equivalent Go type-switch (built
      and run against a reference switch).
- [ ] Bare construction of a cross-`.goal`-package variant emits `pkg.Enum(pkg.Enum_Variant{})`.
- [ ] `task check`, `task build`, `task fixpoint` all green.
- [ ] Corpus behavioral tier unchanged.

## User Interactions

No new CLI surface. Existing `goal build` / `goal run` over a multi-package goal project
now transpiles cross-`.goal`-package enum usage that previously failed.

## Error Handling

An unresolved import remains non-fatal (types left deferred), exactly as today. A sibling
`.goal` package that fails to parse is tolerated (its enums simply do not contribute),
matching the existing tolerance for an unparseable sibling `.go` file.

## Out of Scope

- Projecting struct / function / method foreign facts from sibling `.goal` source
  (enum keystone only; this remains strictly additive — a `.goal`-only dir yields nothing
  today). Those are covered by the `.go` path once a package is emitted, or by later seam
  stories.
- Whole-program / dependency-ordered transpile rearchitecture (the bounded
  enrichment-reads-`.goal` extension suffices).
- Payload-bearing cross-package variant field requalification beyond best-effort (the
  enum types these unblock — FuncMod/ChanDir/Mode/Severity — are tag-only).

## Open Questions

- None. Approach and scope are settled by the SEAM-CAP-2 prd notes and the verified root
  cause; the bounded option is sufficient.
