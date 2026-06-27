# Audit — Completeness

Scope: small, well-bounded parser extension over an existing AST and lexer.

## Findings

- MINOR: The spec does not pin a body that is a block (`Pattern => { ... }`).
  None of the corpus match examples use a block body, but FR-5 already allows
  it; the parser will support `{` → block to be safe. Not blocking.
- MINOR: Qualified enum references (`pkg.Status.Active`) are not present in the
  corpus. The implementation will treat the last dotted segment as the variant
  and the prefix as the enum, which generalizes correctly; no test input
  exercises the qualified form. Not blocking.

No CRITICAL or MAJOR findings. The acceptance criteria map directly to concrete
corpus inputs and are testable.

## Assumptions

- Statement-position match is represented by wrapping the `MatchExpr` in an
  `ExprStmt` (the AST has no separate MatchStmt — confirmed in
  `internal/ast/goal_expr.go`).
- A single payload binding per variant pattern (`Status.Active(a)`); multi-field
  destructuring is out of scope for this story.
