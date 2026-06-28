# Research — US-021 Parse match and patterns

No external research required: this is an internal recursive-descent parser
extension over an established codebase. Findings from reading the code:

- AST is complete for this story (US-016): `ast.MatchExpr` (an `Expr`),
  `ast.MatchArm` (Pattern/Arrow/Body where Body is a `Node`),
  `ast.VariantPattern` (Enum `Expr` + Variant `*Ident` + optional Binding),
  and `ast.RestPattern` already exist in `internal/ast/goal_expr.go`. No AST
  change is needed.
- The lexer already emits `token.MATCH` (reserved keyword) and `token.FAT_ARROW`
  (`=>`, single token, per US-013).
- The parser already establishes the pattern this story follows:
  - Control-clause headers suppress composite-literal braces with `exprLev = -1`
    so a trailing `{` is taken as a body block (see `parseIfStmt`,
    `parseSwitchStmt`). The match subject needs the same treatment so
    `match s {` reads the `{` as the arms brace, not a composite literal.
  - Statement dispatch is `parseStmt`; operand/expression entry is
    `parseOperand` via `parseExpr`. Adding `match` to both gives statement and
    value position from one `parseMatchExpr`.
  - `startsExpr` gates `return <expr>`; add `token.MATCH` so `return match ...`
    parses its result list.
- Inputs to validate against live in `features/02-match/examples/`:
  `status_match.goal` (statement position), `status_return.goal` and
  `status_var.goal` (value position), `status_rest.goal` (rest pattern `_`).

Confidence: High. The grammar is small and the surrounding parser conventions
are well established.
