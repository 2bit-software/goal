# Technical requirements / research — SEAM-004

## Proven capabilities (prerequisites, all passed)
- CAP-3a: sealed interfaces preserve method signatures (Node.Pos()/End() survive).
- CAP-3b: same-package type-pattern match -> Go type-switch with `case *T:` + exhaustiveness.
- CAP-3c: cross-.goal-package sealed implementor-set propagation (foreign.goal).
- CAP-3d: nested sealed hierarchies via embedding cascade — `implements Expr`
  (sealed Expr embeds sealed Node) auto-emits BOTH isExpr() and isNode().

## Syntax
- `sealed interface Node { Pos() token.Pos; End() token.Pos }`
- `sealed interface Expr { Node }`  (embeds Node; cascade handles markers)
- `type Ident struct implements Expr { ... }` (drop the old exprNode() marker method)
- Node-only support types (Field, FieldList, File, DocComment, Variant,
  PayloadField, MatchArm, ImplementsClause) declare `implements Node`.

## Key facts
- A plain type-switch over a sealed interface is NOT rejected by sema, so the seal
  and the switch conversions need not all land in one commit; every committed
  state still builds and the final state has the gates green.
- match value-position lowers only as `:=`/`var x T = match`/`return match`, never
  `field = match`.
- enum/sealed zero value is nil; set sealed fields explicitly at constructors.
- internal/ast (live Go compiler AST) stays open Go; only selfhost/ast seals. The
  shared port-gated test internal/ast/ast_test.go references no marker methods, so
  it compiles against both the open internal/ast and the sealed selfhost transpile.

## Scope decisions
- Convert: sema/{check,resolve,question,fields,mustuse,assert}, backend/{lower,emit},
  parser/{parser,goal_construct}.
- Keep plain (documented): foreign.goal's 3 switches over go/ast (Go stdlib,
  unsealable); emit.goal's sibling-category-interface switches
  (`case ast.Stmt:`/`case ast.Expr:` — a match over *different* sealed interfaces,
  not variants of one); Walk's 60-arm switch (grouped multi-type cases, AC permits
  documented exclusion).

## Gates
task check, task build, task fixpoint green; corpus behavioral + interp tiers unchanged.
