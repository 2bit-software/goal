# Completeness Audit — SEAM-CAP-3b

No CRITICAL or MAJOR findings. The spec covers the happy path (FR-3 lowering),
exhaustiveness error (FR-4), the accept cases (full cover / `_`), and the
behavior-preservation constraint. Acceptance criteria are testable.

## MINOR

- MINOR: The spec exercises pointer implementors (`*T`). Value-type implementor
  patterns are representable by the AST node but not proven. Recorded explicitly
  in Out of Scope — acceptable; the §8.1 / go/ast shape is pointer-based.
- MINOR: Binding semantics: an arm `*T(x)` narrows `x` to the concrete type in the
  body. When subject is itself a simple ident `n` and no explicit binding is given,
  the body referring to `n` is not narrowed. The fixture uses explicit bindings
  where field access is needed, which is unambiguous.

## Assumptions

- A `match` is classified as "sealed" purely by the presence of TypePattern arms,
  not by resolving the scrutinee's static type — this keeps dispatch position-
  independent (matching the enum path which reads arm qualifiers, not the subject).
- Exhaustiveness resolves the sealed interface from the first TypePattern arm's
  concrete type via the implementor registry; mixing implementors of two different
  sealed interfaces in one match is author error and need not be specially
  diagnosed beyond the resulting missing-implementor report.
