# SEAM-CAP-3b: Type-pattern match over a same-package sealed interface — Business Specification

## Overview

goal's `match` today lowers only Result/Option/enum scrutinees, routing every arm
through `Enum_Variant` case labels. This feature adds `match` over a sealed-interface
scrutinee: arms name the concrete implementor types (e.g.
`match n { *Ident => ..., *CallExpr => ... }`) and the construct lowers to a Go
type-switch with concrete `case *T:` labels. This is the language feature that lets
an idiomatic `switch x := n.(type)` become an exhaustive `match`. Scope is a SINGLE
package; cross-package is a separate later story.

## Functional Requirements

### FR-1: Implementor registry
The resolver builds, from `implements` clauses, a registry mapping each sealed
interface to its set of concrete implementor types in the (same) package. It is
unioned across the package's files.

### FR-2: Parser accepts type-pattern arms
The parser accepts a match arm whose pattern is a concrete type (`*T`), optionally
binding the narrowed value (`*T(x)`), as a distinct pattern from an enum variant
pattern. Existing enum/Result/Option patterns are unaffected.

### FR-3: Backend sealedMatch lowering
A `match` over a sealed-interface scrutinee lowers via a separate path (distinct
from the enum path) to a Go type-switch: `switch [guard :=] subject.(type)` with one
`case *T:` per arm, a `_` rest-arm becoming `default`, and a proven-exhaustive match
(no rest) a panicking default — mirroring the enum lowering's shape. Works in
statement, `return`, and `var x T =` positions.

### FR-4: Sema exhaustiveness over the registry
A `match` over a sealed interface must cover every registered implementor or carry a
`_` rest-arm; otherwise it is an Error naming the missing implementors — consistent
with the §9 switch-coexistence rule (a plain switch on a sealed value is itself an
error, so the match must be the exhaustive replacement).

### FR-5: Mirrored in internal/ and selfhost/
All four changes land in BOTH the live Go transpiler (internal/) and the
self-hosted .goal mirror (selfhost/), so `task fixpoint` stays green.

## Acceptance Criteria

- [ ] A same-package `match` over a sealed interface transpiles to a Go type-switch
      with concrete `case *T:` labels.
- [ ] The transpiled program behaves identically to the equivalent
      `switch x := n.(type)`.
- [ ] A non-exhaustive `match` over a sealed interface (missing an implementor, no
      `_`) is a sema Error.
- [ ] An exhaustive match, or one with a `_` rest-arm, is accepted.
- [ ] Existing enum/Result/Option match, the corpus behavioral tier, and the
      self-host fixpoint are unchanged.
- [ ] `task check`, `task build`, `task fixpoint` all green.

## User Interactions

A goal author writes:
```
match n {
    *Lit(l) => l.Val,
    *Neg(g) => -eval(g.Inner),
}
```
over a value `n` of a sealed-interface type, in statement, return, or var position.

## Error Handling

- Non-exhaustive sealed match without `_`: Error `non-exhaustive-match`
  (feature 02-match) naming the uncovered implementor types.
- A sealed interface whose implementors cannot be resolved in-file: deferred (no
  false rejection) — the same defer discipline as the enum path.

## Out of Scope

- Cross-.goal-package sealed-interface match (implementor propagation across
  packages) — that is SEAM-CAP-3c.
- Converting selfhost's existing AST type-switches to match — that is SEAM-004.
- Value-type (non-pointer) implementor patterns beyond what the type renderer
  already supports; the AST node is general but the proof exercises pointer
  implementors (the §8.1 / go/ast shape).

## Open Questions

None blocking.
