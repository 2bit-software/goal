# Technical Requirements & Research — US-016

## Constraints

- Zero-dependency: stdlib only; tests use stdlib `testing` (no testify).
- Follow REWRITE-ARCHITECTURE.md §1.3 (lines 467-475): the goal expression and
  pattern node set.
- Node design mirrors go/ast trimmed to goal's subset, carrying our own
  `token.Pos` (Offset/Line/Col) — positions live on the tree.

## Design (mirrors existing internal/ast conventions)

New file `internal/ast/goal_expr.go`, alongside `goal_decl.go` (US-015):

- `MatchExpr` (Expr): `match Subject { Arms }` — usable in value and statement
  position. Fields: Match pos, Subject Expr, Lbrace, Arms []*MatchArm, Rbrace.
- `MatchArm` (support Node): Pattern Expr, Arrow pos (`=>`), Body Node.
- `VariantPattern` (Expr, a destructuring pattern): Enum Expr (Ident/Selector),
  Variant *Ident, Lparen, Binding *Ident (payload bind), Rparen.
- `RestPattern` (Expr): the `_` catch-all arm pattern; Underscore pos.
- `UnwrapExpr` (Expr): postfix `?`; X Expr, Question pos.
- `VariantLit` (Expr, a construction): Enum Expr, Variant *Ident, Lparen,
  Args []Expr (LabeledArg or positional), Rparen.
- `LabeledArg` (Expr): Label *Ident, Colon pos, Value Expr — `since: now()`.
- `SpreadElement` (Expr): Ellipsis pos, X Expr — `...defaults` / `...derive(s)`.

Each gets the right category marker (exprNode for the expression/pattern forms;
MatchArm is a plain support Node like Field/Variant). Walk grows a case per
node. The test follows the US-015 `TestWalkGoalDeclChildren` collect+assertChildren
pattern and adds the distinct-node-type assertion required by AC 2.

## Verify

- prd.json verifyCommands: `go build ./...`, `go vet ./...`,
  `go test ./... -count=1`.
- Plus: `go test ./internal/ast/ -run TestWalkGoalExpr` for the new test.
