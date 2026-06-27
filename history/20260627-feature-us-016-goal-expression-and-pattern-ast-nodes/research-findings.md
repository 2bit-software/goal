# Research Findings — US-016

This is an internal AST node-definition story; the authoritative sources are
in-repo, not external. Findings:

## 1. Node set required (REWRITE-ARCHITECTURE.md §1.3, lines 467-475)

Expressions: `MatchExpr` (as an *expression*), `MatchArm`,
`VariantPattern`/`RestPattern`, `VariantLit` with `LabeledArg`, `UnwrapExpr`
for postfix `?`, `SpreadElement` for `...defaults`/`...derive`. The doc stresses
(lines 154-160) that the three meanings of `Enum.Variant(x)` — construct /
destructure-bind / ordinary call — must be *distinct node types*, which is
exactly AC 2.

## 2. Existing conventions to mirror (internal/ast)

- `goal_decl.go` (US-015) is the template: a separate file declares the new
  nodes; each carries `token.Pos` fields; Pos()/End() computed from positions;
  category markers are unexported methods (`exprNode()` / `declNode()`).
- Support nodes (Field, Variant, PayloadField, ImplementsClause) implement only
  `Node` — no category marker — and still walk fine because Walk takes `Node`.
  `MatchArm` follows this (like `Variant`).
- `walk.go` has an open type switch; new nodes add a case each. `walkExpr`,
  `walkExprList`, `walkIdentList` skip nil children.
- Tests are `package ast`, stdlib `testing` only. `TestWalkGoalDeclChildren`
  gives the collect()+assertChildren() pattern to reuse for the new test.

## 3. Concrete goal syntax (from features/*/examples/*.goal)

- Construction: `Status.Active(since: now())` -> VariantLit{Enum:Status,
  Variant:Active, Args:[LabeledArg{since, now()}]}.
- Destructure in match arm: `Status.Active(a) => render(a.since)` ->
  MatchArm{Pattern: VariantPattern{Enum:Status, Variant:Active, Binding:a},
  Body: render(a.since)}.
- Rest arm: `_ => showPlaceholder()` -> MatchArm{Pattern: RestPattern, ...}.
- Data-less arm: `Status.Pending => startOnboarding()` -> VariantPattern with
  no Lparen/Binding.
- Spread: `User{name: name, ...defaults}` -> CompositeLit Elts containing a
  SpreadElement{X: Ident "defaults"}; `...derive(s)` -> SpreadElement{X: Call}.

## Confidence: High — all inputs are in-repo and unambiguous.

## Open questions: none. No external dependencies; node-only (no parsing/lowering).
