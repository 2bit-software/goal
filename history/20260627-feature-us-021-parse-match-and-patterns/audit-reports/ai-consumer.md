# Audit — AI-Consumer Readiness

## Findings

- The spec references concrete, existing inputs (`features/02-match/examples/*`)
  and existing AST node types, so an implementer can write test assertions
  directly without guessing field names.
- All terms (match, arm, variant pattern, rest pattern, binding) are defined by
  the existing AST nodes in `internal/ast/goal_expr.go`.
- State transitions are explicit: statement vs value position both produce a
  `MatchExpr`; statement position wraps it in `ExprStmt`.

No CRITICAL or MAJOR findings.

## Assumptions

- `token.MATCH` and `token.FAT_ARROW` already exist and are emitted by the lexer
  (verified).
- The subject expression is parsed with composite-literal braces suppressed
  (`exprLev = -1`) so `match s {` reads `{` as the arms brace — same convention
  the parser already uses for `if`/`for`/`switch` headers.
