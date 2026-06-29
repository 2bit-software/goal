# SEAM-CAP-3c — cross-.goal-package sealed-interface match — Business Specification

## Overview

A `sealed interface` defined in one `.goal` package must be matchable (via type-pattern
`match`) from a consumer `.goal` package during the real per-package
`goal build ./selfhost` bootstrap, behaving identically to the equivalent Go type-switch,
with exhaustiveness verified across the package boundary. This is the final prerequisite
for SEAM-004, 35 of whose 36 type-switches consume AST node interfaces from packages
other than where those interfaces are defined.

Today foreign enrichment propagates enum facts from sibling `.goal` source but not
sealed-interface implementor sets, so a cross-package type-pattern match's exhaustiveness
cannot be resolved (it defers with a warning).

## Functional Requirements

### FR-1: Implementor-set propagation
Foreign enrichment projects a sibling-.goal-package sealed interface's implementor set
(derived from its `implements` clauses) into the consuming package's sealed-interface
registry, keyed by the qualified interface name, with implementors named as the consumer
sees them (qualified concrete pointer types).

### FR-2: Cross-package match resolves and lowers
A cross-package type-pattern `match` over such a sealed interface resolves to its defining
interface and lowers to a Go type-switch with concrete qualified `case` labels.

### FR-3: Cross-package exhaustiveness
A complete cross-package match (covering every projected implementor, or with a `_` rest)
produces no diagnostic; a non-exhaustive one (missing an implementor, no `_`) is a
`non-exhaustive-match` error naming the missing implementor — not the
`unresolved-match-sealed` deferral.

### FR-4: Behavioral equivalence in the real topology
With both the defining and consuming packages transpiled per-package and built, the
lowered match behaves identically to a hand-written `switch x := n.(type)` over the same
implementor types.

## Acceptance Criteria

- [ ] Foreign enrichment propagates a sibling-.goal-package sealed interface's implementor
      set (extending the SEAM-CAP-2 goal-source reading path that today handles enums only).
- [ ] Implemented in BOTH the live transpiler and the self-host `.goal` mirror.
- [ ] A 2+-package fixture proves a `match` over a sealed interface defined in a SIBLING
      `.goal` package transpiles and behaves identically to the type-switch in the real
      build topology.
- [ ] A cross-package non-exhaustive match is a `non-exhaustive-match` error; a complete
      one is clean (no `unresolved-match-sealed` warning).
- [ ] `task check`, `task build`, `task fixpoint` all green; corpus behavioral tier unchanged.

## User Interactions

None directly user-facing. The interface is the goal language: authors write a cross-package
`match` over an imported sealed interface and it type-checks/transpiles like a same-package one.

## Error Handling

- A non-exhaustive cross-package sealed match is a compile-time `non-exhaustive-match` error.
- An unresolvable sealed type (e.g. a package that cannot be read) still defers with the
  existing `unresolved-match-sealed` warning rather than a false rejection.

## Out of Scope

- SEAM-004's actual sealing of ast.Node/Expr/Stmt/Decl/Spec and conversion of the ~43
  type-switches.
- Reconstructing sealed-interface implementor sets from a sibling's GENERATED `.go`
  (the `.go` foreign path) — the real bootstrap resolves siblings to `.goal` source;
  this mirrors SEAM-CAP-2 deferring struct/func/method `.goal` facts.
- Propagating non-sealed `implements` relations across packages.

## Open Questions

None. The approach reuses the SEAM-CAP-2 machinery and the CAP-3b registry shape.
