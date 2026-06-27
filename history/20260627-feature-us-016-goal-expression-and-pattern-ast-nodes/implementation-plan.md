# Implementation Plan — US-016 Goal expression and pattern AST nodes

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/ast/goal_expr.go` | Declares goal's expression and pattern nodes (MatchExpr, MatchArm, VariantPattern, RestPattern, UnwrapExpr, VariantLit, LabeledArg, SpreadElement) with token.Pos fields and Pos()/End()/marker methods, paralleling goal_decl.go. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/ast/walk.go` | Add one Walk type-switch case per new node, descending into children via Walk/walkExpr/walkExprList. |
| `internal/ast/ast_test.go` | Add TestWalkGoalExprChildren: build each new node, assert Walk descends into children exactly once (reuse collector/assertChildren), and assert a construction VariantLit and a destructuring VariantPattern are distinct node types. |

## Package Structure

```
internal/ast/
  ast.go          (Go subset — unchanged)
  goal_decl.go    (US-015 goal decls — unchanged)
  goal_expr.go    (NEW — US-016 goal exprs/patterns)
  walk.go         (MODIFIED — new cases)
  ast_test.go     (MODIFIED — new test)
```

## Dependency Graph

1. `goal_expr.go` node types — depend only on existing `token` and the Expr/Node
   markers already in `ast.go`.
2. `walk.go` cases — depend on (1).
3. `ast_test.go` test — depends on (1) and (2).

## Interface Contracts

```go
// MatchExpr is `match Subject { Arms }` — an expression (value or stmt position).
type MatchExpr struct {
    Match   token.Pos
    Subject Expr
    Lbrace  token.Pos
    Arms    []*MatchArm
    Rbrace  token.Pos
}
func (e *MatchExpr) Pos() token.Pos
func (e *MatchExpr) End() token.Pos
func (*MatchExpr) exprNode() {}

// MatchArm is one `Pattern => Body` arm (support Node, like Variant).
type MatchArm struct {
    Pattern Expr
    Arrow   token.Pos // position of "=>"
    Body    Node
}
func (a *MatchArm) Pos() token.Pos
func (a *MatchArm) End() token.Pos

// VariantPattern is a destructuring pattern: Enum.Variant(Binding).
type VariantPattern struct {
    Enum    Expr      // Ident or SelectorExpr; may be nil for a bare variant
    Variant *Ident
    Lparen  token.Pos // zero for a data-less pattern
    Binding *Ident    // payload bind; or nil
    Rparen  token.Pos
}
func (p *VariantPattern) Pos() token.Pos
func (p *VariantPattern) End() token.Pos
func (*VariantPattern) exprNode() {}

// RestPattern is the `_` catch-all arm pattern.
type RestPattern struct {
    Underscore token.Pos
}
func (p *RestPattern) Pos() token.Pos
func (p *RestPattern) End() token.Pos
func (*RestPattern) exprNode() {}

// UnwrapExpr is postfix `?`.
type UnwrapExpr struct {
    X        Expr
    Question token.Pos
}
func (e *UnwrapExpr) Pos() token.Pos
func (e *UnwrapExpr) End() token.Pos
func (*UnwrapExpr) exprNode() {}

// VariantLit is a construction: Enum.Variant(Args).
type VariantLit struct {
    Enum    Expr
    Variant *Ident
    Lparen  token.Pos // zero for a data-less construction
    Args    []Expr    // LabeledArg and/or positional
    Rparen  token.Pos
}
func (e *VariantLit) Pos() token.Pos
func (e *VariantLit) End() token.Pos
func (*VariantLit) exprNode() {}

// LabeledArg is a `Label: Value` argument.
type LabeledArg struct {
    Label *Ident
    Colon token.Pos
    Value Expr
}
func (e *LabeledArg) Pos() token.Pos
func (e *LabeledArg) End() token.Pos
func (*LabeledArg) exprNode() {}

// SpreadElement is `...X` inside a composite literal.
type SpreadElement struct {
    Ellipsis token.Pos
    X        Expr
}
func (e *SpreadElement) Pos() token.Pos
func (e *SpreadElement) End() token.Pos
func (*SpreadElement) exprNode() {}
```

## Integration Points

- `internal/ast/walk.go` `Walk` switch: add cases mirroring existing ones —
  `*MatchExpr` (Subject, each Arm), `*MatchArm` (Pattern, Body), `*VariantPattern`
  (Enum, Variant, Binding), `*RestPattern` (no children), `*UnwrapExpr` (X),
  `*VariantLit` (Enum, Variant, each Arg), `*LabeledArg` (Label, Value),
  `*SpreadElement` (X). Use `walkExpr` for optional Expr children and guard nil
  *Ident / nil Node children like the existing cases.
- No other package references these nodes yet (parser/lowering are later stories).

## Testing Strategy

- `internal/ast/ast_test.go`, package `ast`, stdlib `testing` only.
- New `TestWalkGoalExprChildren` reuses the existing `collector` plus a local
  `collect`/`assertChildren` (as in `TestWalkGoalDeclChildren`).
- Build: a MatchExpr with a VariantPattern arm and a RestPattern arm; a VariantLit
  with a LabeledArg; an UnwrapExpr; a SpreadElement. Assert parent + each child
  visited exactly once.
- Distinct-type assertion: construct a VariantLit and a VariantPattern of the same
  surface (`Status.Active(...)`) and assert `fmt.Sprintf("%T", lit) !=
  fmt.Sprintf("%T", pat)` and that each is the expected concrete type.
- Verify with prd.json gates: `go build ./...`, `go vet ./...`,
  `go test ./... -count=1`, plus `go test ./internal/ast/`.
