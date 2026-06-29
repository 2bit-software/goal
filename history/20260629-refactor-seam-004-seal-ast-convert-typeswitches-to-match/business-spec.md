# Business spec — SEAM-004

## Outcome
The goal compiler's central data structure (the AST) showcases goal's exhaustive
`match` idiom rather than open Go interfaces + plain type-switches.

## Requirements
- ast.Node, Expr, Stmt, Decl, Spec are sealed interfaces (§8.1 encoding).
- AST-family type-switches are converted to `match` with exhaustiveness.
- Behavior is preserved: the self-hosted compiler still compiles itself to a
  byte-identical fixpoint and the corpus behaves identically.
- The go/ast-mirror oracle tension (selfhost/ast modeled on go/ast) is resolved
  with a documented decision.

## Constraints (relaxed seam gate)
- Emitted Go MAY change; equivalence is re-proven by `task fixpoint`
  self-consistency + corpus behavioral tier + reviewed golden regeneration.
