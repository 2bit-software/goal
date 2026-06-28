# Plan Audit — Buildability

- Dependency order is valid: AST exists; parsing methods depend only on existing
  `parseExpr`/`parseBlock`/`exprLev`; dispatch hooks depend on the methods;
  tests last. No forward references.
- Interface contracts agree: `parseMatchExpr` returns `*ast.MatchExpr` (an
  `Expr`), usable directly as `parseOperand`'s return and wrappable in
  `ast.ExprStmt` for `parseStmt`. `MatchArm.Body` is `Node`, accepting either a
  `*ast.BlockStmt` or an `Expr`.
- File paths verified: `internal/parser/` exists; `goal_match.go` /
  `goal_match_test.go` do not yet exist (no conflict).
- Integration points are specific (file + function + how).

No CRITICAL/MAJOR findings.

## Assumptions

- The match subject needs `exprLev = -1` to avoid reading `match s {` as a
  composite literal — same convention as existing control-clause headers.
- `_` lexes as an `IDENT` whose `Lit` is `"_"` (standard identifier lexing).
