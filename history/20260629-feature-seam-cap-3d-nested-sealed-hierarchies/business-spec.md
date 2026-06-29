# Nested sealed-interface hierarchies — Business Specification

## Overview

The goal language supports sealed interfaces (a closed set of concrete
implementors named via `implements` clauses) with exhaustive `match`. Until now
only FLAT, single-level sealed interfaces worked. This feature adds 2-level
hierarchies: a sealed interface B may embed a sealed interface A, and a concrete
type declared `implements B` is treated as an implementor of BOTH A and B. This
unblocks sealing the self-hosted AST, whose category interfaces (Expr, Stmt,
Decl, Spec) all embed Node.

## Functional Requirements

### FR-1: Marker satisfaction cascades through embedded sealed interfaces
When sealed interface B embeds sealed interface A, a concrete type T declared
`implements B` SHALL satisfy both A and B in the emitted Go. The transpiled
package SHALL `go build` cleanly with no "missing method isA" error.

### FR-2: Implementor registration cascades for exhaustiveness
T SHALL be registered as an implementor of both A and B. A `match` over A SHALL
include T, and a `match` over B SHALL include T.

### FR-3: Exhaustiveness enforced at both levels
A `match` over A that omits T (and has no rest arm) SHALL be a sema error; the
same SHALL hold for a `match` over B. A `match` covering all implementors SHALL
be accepted.

### FR-4: Behavior parity with type-switch
A `match` over A or B SHALL behave identically to the equivalent hand-written Go
`switch x := n.(type)` for the same values.

### FR-5: Transitivity
The cascade SHALL be transitive: if C embeds B and B embeds A (all sealed), an
implementor of C is an implementor of A, B, and C.

## Acceptance Criteria

- [ ] A 2-level sealed hierarchy (sealed B embeds sealed A; concrete T implements
      B) transpiles and `go build`s cleanly.
- [ ] `match` over A includes T and `match` over B includes T; both equal the
      reference type-switch for every value.
- [ ] A non-exhaustive `match` over A is a sema error; a non-exhaustive `match`
      over B is a sema error.
- [ ] Existing flat sealed interfaces are unaffected (behavior-preserving).
- [ ] Fixed in both internal/ (live transpiler) and selfhost/ (.goal mirror).
- [ ] task check, task build, task fixpoint all green; corpus behavioral unchanged.

## User Interactions

Goal source authors write:
```
sealed interface A {}
sealed interface B { A }          // B embeds A
type T struct implements B { ... }
match x { *T(t) => ... }          // over A or over B
```
No new syntax — uses existing `sealed interface`, embedding, and `implements`.

## Error Handling

A non-exhaustive match over either level produces the existing
`non-exhaustive-match` diagnostic. An unresolved sealed type continues to defer
with the existing `unresolved-match-sealed` warning.

## Out of Scope

- Multi-interface `implements B, A` clause syntax (cascade chosen instead).
- Sealing the actual AST and converting its type-switches (that is SEAM-004).
- Cross-package nested hierarchies beyond what CAP-3c already propagates (the AST
  is a single package; the cascade rides through the existing foreign projection).

## Open Questions

None. Design (embedding cascade) is settled by the existing CAP-3a/b/c machinery.
