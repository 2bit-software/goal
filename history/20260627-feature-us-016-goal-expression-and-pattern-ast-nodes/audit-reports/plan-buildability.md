# Plan Audit: Buildability — US-016

## Findings

### None CRITICAL / None MAJOR

- Dependency order is a valid topological sort: node types (1) -> Walk cases (2)
  -> test (3). No forward references; nodes depend only on existing `token` and
  the Expr/Node markers in ast.go.
- Interface contracts are concrete Go signatures matching existing conventions
  (token.Pos fields, Pos()/End(), unexported marker). They compile in isolation:
  each Expr node carries exprNode(); MatchArm is a plain Node.
- File paths are real and verified: `internal/ast/goal_expr.go` (new, parallels
  `goal_decl.go`), `internal/ast/walk.go`, `internal/ast/ast_test.go` all exist
  / are creatable.
- Integration point is specific: the named cases in walk.go's `Walk` type switch,
  using the existing `walkExpr`/`walkExprList` nil-skipping helpers.

### MINOR — Walk for *MatchArm.Body must use Walk (Node), not walkExpr
`walkExpr` takes an `Expr`; Body is `Node`, so the case must call `Walk(v, n.Body)`
guarded by a nil check (a nil interface Node). Noted so the implementer uses the
right helper. Not a blocker — the existing DeclStmt/Defer cases show the pattern.

## Assumptions

- New nodes are not yet referenced by any other package, so no cross-package
  integration is needed this story.
- The test file gains one new test function; the existing two tests are untouched.
