# Implementation Tasks — US-016

## Task 1: Declare goal expression and pattern nodes
**Status**: completed
**Files**: `internal/ast/goal_expr.go` (new)
**Depends on**: (none)
**Spec coverage**: FR-1..FR-5 (node existence); AC 1
**Verify**: `go build ./internal/ast/`

### Instructions
- Create `internal/ast/goal_expr.go`, `package ast`, importing
  `goal/internal/token`, paralleling `goal_decl.go`.
- Declare structs with token.Pos fields and Pos()/End() methods, plus the
  category marker, exactly per the implementation-plan interface contracts:
  - `MatchExpr` (exprNode): Match, Subject Expr, Lbrace, Arms []*MatchArm, Rbrace.
    Pos = Match; End = Rbrace+1.
  - `MatchArm` (support Node only): Pattern Expr, Arrow token.Pos, Body Node.
    Pos = Pattern.Pos() (fallback Arrow); End = Body.End() (fallback Arrow+2).
  - `VariantPattern` (exprNode): Enum Expr, Variant *Ident, Lparen, Binding
    *Ident, Rparen. Pos = Enum.Pos() else Variant.Pos(); End = Rparen+1 when set,
    else Variant.End() / Binding.End().
  - `RestPattern` (exprNode): Underscore token.Pos. Pos = Underscore;
    End = Underscore+1.
  - `UnwrapExpr` (exprNode): X Expr, Question token.Pos. Pos = X.Pos();
    End = Question+1.
  - `VariantLit` (exprNode): Enum Expr, Variant *Ident, Lparen, Args []Expr,
    Rparen. Pos = Enum.Pos() else Variant.Pos(); End = Rparen+1 when set, else
    Variant.End().
  - `LabeledArg` (exprNode): Label *Ident, Colon, Value Expr. Pos = Label.Pos();
    End = Value.End() else Colon+1.
  - `SpreadElement` (exprNode): Ellipsis token.Pos, X Expr. Pos = Ellipsis;
    End = X.End() else Ellipsis+3.
- Guard nil fields when computing Pos/End, mirroring goal_decl.go (compare
  against `token.Pos{}` for optional position fields).
- Add a file-level doc comment explaining these are goal's expression/pattern
  surface and that they make the three meanings of `Enum.Variant(x)` distinct
  node types.

## Task 2: Wire new nodes into Walk
**Status**: completed
**Files**: `internal/ast/walk.go` (modify)
**Depends on**: Task 1
**Spec coverage**: FR-6; AC 3
**Verify**: `go build ./internal/ast/`

### Instructions
- Add a "Goal expressions / patterns" section of cases to the `Walk` type switch:
  - `*MatchExpr`: walkExpr(v, n.Subject); for each arm `Walk(v, arm)`.
  - `*MatchArm`: walkExpr(v, n.Pattern); if n.Body != nil `Walk(v, n.Body)`.
  - `*VariantPattern`: walkExpr(v, n.Enum); if n.Variant != nil Walk; if
    n.Binding != nil Walk.
  - `*RestPattern`: no children.
  - `*UnwrapExpr`: walkExpr(v, n.X).
  - `*VariantLit`: walkExpr(v, n.Enum); if n.Variant != nil Walk; walkExprList(v,
    n.Args).
  - `*LabeledArg`: if n.Label != nil Walk; walkExpr(v, n.Value).
  - `*SpreadElement`: walkExpr(v, n.X).
- Body is a `Node`, so use `Walk(v, n.Body)` guarded by nil, not walkExpr.

## Task 3: Test distinct node types and Walk descent
**Status**: completed
**Files**: `internal/ast/ast_test.go` (modify)
**Depends on**: Task 1, Task 2
**Spec coverage**: AC 1, AC 2, AC 3
**Verify**: `go test ./internal/ast/ -run TestWalkGoalExpr -count=1`

### Instructions
- Add `TestWalkGoalExprChildren`, package `ast`, reusing the existing `collector`
  and the collect()/assertChildren() helper pattern from
  `TestWalkGoalDeclChildren`.
- Build, with real nodes:
  - A VariantLit `Status.Active(since: now())`: Enum Ident "Status", Variant
    "Active", Args [LabeledArg{Label "since", Value CallExpr now()}].
  - A VariantPattern `Status.Active(a)`: Enum Ident "Status", Variant "Active",
    Binding "a".
  - A MatchExpr with two arms: arm1 Pattern = the VariantPattern, Body =
    CallExpr; arm2 Pattern = RestPattern, Body = CallExpr.
  - An UnwrapExpr wrapping a CallExpr.
  - A SpreadElement wrapping Ident "defaults".
- Assert Walk descends into each node's children exactly once (assertChildren).
- Assert distinct types: `fmt.Sprintf("%T", variantLit) != fmt.Sprintf("%T",
  variantPattern)` and each equals its expected concrete type string.
- Keep stdlib `testing`/`fmt` only (no testify).

## Task 4: Full verify
**Status**: completed
**Files**: (none)
**Depends on**: Task 1-3
**Spec coverage**: AC 4
**Verify**: `go build ./...` && `go vet ./...` && `go test ./... -count=1`
