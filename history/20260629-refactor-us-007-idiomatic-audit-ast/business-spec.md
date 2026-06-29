# US-007 Idiomatic audit: ast — Business Specification

## Overview

Step 3 (idiomatic audit) of the SELF-HOST IDIOMATIC plan, applied to the
highest idiomatic-opportunity package: `selfhost/ast`. The audit evaluates the
package's Go-isms — the node-kind category interfaces (Node/Decl/Stmt/Expr/Spec),
the iota integer enums (FuncMod, ChanDir), and the `Walk` type-switch over node
kinds — against the goal idioms they could become (`sealed interface`, `enum`,
`match`). Each candidate is converted where the idiom genuinely fits and is
behavior-preserving, or refused with a recorded rationale in DECISIONS.md. The
package must keep behaving byte-for-byte identically: the US-003 verbatim
self-host is the behavioral oracle.

## Functional Requirements

### FR-1: Node-kind idiom evaluation
Every node-kind group expressed as a Go interface, and every iota-based node-kind
enumeration, SHALL be evaluated for `sealed interface`/`enum` representation. Any
group that fits the goal idiom AND is behavior-preserving SHALL be converted; any
group that does not SHALL be left as-is with a refusal-with-reason recorded in
DECISIONS.md.

### FR-2: switch-over-node-kind to match
Each `switch`/type-switch over a node kind SHALL be converted to `match` where it
fits (i.e. where the scrutinee is a closed enum/sealed interface). Where it does
not fit, the decision SHALL be recorded.

### FR-3: No remaining auto-convertible propagation
After the audit, `goal fix` SHALL report no remaining auto-convertible
propagation sites in `selfhost/ast`.

### FR-4: Behavior preservation
The transpiled `selfhost/ast` SHALL keep the same public API the US-003 oracle
pins; the ported ast tests SHALL pass against it; and `task fixpoint` SHALL stay
byte-identical green.

## Acceptance Criteria

- [ ] Node-kind groups expressed as Go interfaces are evaluated for `sealed
      interface`/`enum` representation, with conversions or a recorded
      DECISIONS.md rationale; switch-over-node-kind becomes `match` where it fits.
- [ ] `goal fix` reports no remaining auto-convertible propagation sites in the
      package.
- [ ] ast tests pass against the transpiled package and `task fixpoint` stays
      green.
- [ ] `task check` and `task build` are green.

## User Interactions

None — this is an internal compiler-source audit. The observable surface is the
DECISIONS.md ledger entry and the unchanged behavior of the goal-built compiler.

## Error Handling

Not applicable — no runtime behavior is added or changed. The package has no
error-returning functions; it is pure AST data plus a total `Walk` traversal.

## Out of Scope

- Editing any package other than `selfhost/ast` (consumers in selfhost/sema,
  selfhost/backend, selfhost/parser are NOT touched by this story).
- Any change to a public signature or type the US-003 oracle pins.
- Result/`?`/Option conversions (none exist in this package to convert).

## Open Questions

None. The audit is fully determined by the package contents and the established
US-005/US-006 audit pattern.
