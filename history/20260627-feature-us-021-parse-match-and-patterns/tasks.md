# Implementation Tasks

## Task 1: Add match/pattern parsing methods and dispatch
**Status**: completed
**Files**: `internal/parser/goal_match.go` (new), `internal/parser/parser.go` (modify)
**Depends on**: (none — AST already exists)
**Spec coverage**: FR-1..FR-5
**Verify**: `go build ./... && go vet ./...`

### Instructions
- New file `internal/parser/goal_match.go`, `package parser`:
  - `parseMatchExpr() *ast.MatchExpr`: expect `token.MATCH`; save `exprLev`, set
    `exprLev = -1`, parse subject via `parseExpr`, restore `exprLev`; expect
    `LBRACE`; loop `parseMatchArm()` until `RBRACE`/`EOF`; expect `RBRACE`.
  - `parseMatchArm() *ast.MatchArm`: `parsePattern()`; expect `FAT_ARROW`; body =
    `parseBlock()` when at `LBRACE`, else `parseExpr()`.
  - `parsePattern() ast.Expr`: if at `IDENT` with `Lit == "_"` → `*ast.RestPattern`;
    else `parseVariantPattern()`.
  - `parseVariantPattern() ast.Expr`: read first ident as Variant; while at
    `PERIOD`, fold prior Variant into Enum (`*Ident` first, then `*SelectorExpr`)
    and set Variant to the next ident; optional `LPAREN IDENT? RPAREN` → Binding.
- Modify `internal/parser/parser.go`:
  - `parseStmt` switch: `case token.MATCH: return &ast.ExprStmt{X: p.parseMatchExpr()}`.
  - `parseOperand` switch: `case token.MATCH: return p.parseMatchExpr()`.
  - `startsExpr`: add `token.MATCH`.

## Task 2: Add match parse tests
**Status**: completed
**Files**: `internal/parser/goal_match_test.go` (new)
**Depends on**: Task 1
**Spec coverage**: all acceptance criteria
**Verify**: `go test ./internal/parser/ -run Match -count=1`

### Instructions
- `package parser`, stdlib `testing` only. Reuse `readExample` from
  `goal_decl_test.go` (same package).
- Statement position: parse `features/02-match/examples/status_match.goal`;
  assert no error; find `handle` FuncDecl; assert `Body.List[0]` is `*ast.ExprStmt`
  wrapping `*ast.MatchExpr` with 3 arms; assert arm 0 pattern variant `Pending`,
  arm 1 `Active` with Binding `a`, arm 2 `Cancelled` with Binding `c`.
- Value position: parse `status_var.goal` (var initializer) and
  `status_return.goal` (return result); locate the `*ast.MatchExpr`; assert 3
  arms, no error.
- Rest pattern: parse `status_rest.goal`; assert last arm pattern is
  `*ast.RestPattern`.

## Task 3: Full verify gates
**Status**: completed
**Files**: (none)
**Depends on**: Task 2
**Spec coverage**: verify-gate acceptance criterion
**Verify**: `go build ./... && go vet ./... && go test ./... -count=1`
