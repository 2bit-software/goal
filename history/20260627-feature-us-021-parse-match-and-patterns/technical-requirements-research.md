# Technical Requirements / Research — US-021

## Existing seams

- AST nodes already exist (US-016): `ast.MatchExpr`, `ast.MatchArm`,
  `ast.VariantPattern`, `ast.RestPattern` in `internal/ast/goal_expr.go`.
  `MatchExpr` is an `Expr` (so it works in value position); statement-position
  match reuses the same node wrapped in an `ast.ExprStmt` (no separate
  MatchStmt).
- Lexer emits `token.MATCH` (reserved keyword) and `token.FAT_ARROW` (`=>`, one
  token).
- Parser (`internal/parser/parser.go`) dispatches statements in `parseStmt` and
  operands in `parseOperand`; expression entry is `parseExpr`.

## Plan

- Add `parseMatchExpr()` returning `*ast.MatchExpr`: parse the `match` keyword,
  the subject expression with composite-literal braces suppressed
  (`exprLev = -1`, like control-clause headers) so the body `{` is the arms
  brace, then the arms until `}`.
- Add `parseMatchArm()`: pattern, `=>`, body. Body is a block when it starts
  with `{`, otherwise an expression.
- Add pattern parsing: `_` → `RestPattern`; otherwise a dotted name where the
  last segment is the variant and the prefix is the Enum (`*Ident` or
  `*SelectorExpr`), with an optional `(binding)`.
- Dispatch: `parseStmt` case `token.MATCH` → `ExprStmt{X: parseMatchExpr()}`;
  `parseOperand` case `token.MATCH` → `parseMatchExpr()`. Add `token.MATCH` to
  `startsExpr` so `return match ...` parses its result.

## Tests

Use the `features/02-match/examples` inputs: `status_match.goal`
(statement position) and `status_var.goal` / `status_return.goal` (value
position). Assert arm counts and pattern/binding/rest structure.
