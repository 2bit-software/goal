# Technical plan — SEAM-004

## Phase 1 — Seal the AST (selfhost/ast/*.goal)
1. ast.goal: convert `Node`, `Decl`, `Stmt`, `Expr`, `Spec` from open interfaces
   to sealed interfaces. Node keeps `Pos()/End()`; Decl/Stmt/Expr/Spec become
   `sealed interface X { Node }`.
2. Drop every marker method (`declNode/stmtNode/exprNode/specNode`) across
   ast.goal, goal_decl.goal, goal_expr.goal, goal_stmt.goal.
3. Add `implements <Category>` to every concrete category type.
4. Add `implements Node` to Node-only support types: Field, FieldList, File,
   DocComment, Variant, PayloadField, MatchArm, ImplementsClause.
- Expected emit: `type X interface { ...; isX() }` + per-type `isX()`/`isNode()`
  cascade. Plain type-switches keep lowering verbatim, so the build stays green
  after this phase alone (sema does not reject plain switch over sealed).
- Verify: task check, task build, task fixpoint.

## Phase 2 — Convert AST-family type-switches to `match`
Per-file, each kept green. `_` rest-arm where a switch handles a subset.
- selfhost/sema: check.goal, resolve.goal, question.goal, fields.goal,
  mustuse.goal, assert.goal.
- selfhost/backend: lower.goal, emit.goal (only the over-one-sealed-interface
  switches; sibling-category switches stay plain).
- selfhost/parser: parser.goal, goal_construct.goal.
- Value-position matches lowered as `:=`/`var`/`return`, never `field = match`.

## Kept plain (documented in DECISIONS.md)
- foreign.goal: 3 switches over go/ast (Go stdlib) — unsealable.
- emit.goal: switches over SIBLING category interfaces (`case ast.Stmt/Expr:`) —
  a discriminator across different sealed interfaces, not variants of one.
- walk.goal: 60-arm switch with grouped multi-type cases — AC permits documented
  exclusion; converting would need empty/grouped arms.

## Phase 3 — Oracle + docs
- DECISIONS.md: resolve the go/ast-mirror tension (mirror tests reference no
  marker methods, so they survive the seal unchanged; selfhost/ast no longer
  needs to mirror go/ast's open-interface shape). Record kept-plain switches.

## Gates
task check, task build, task fixpoint green; corpus behavioral + interp unchanged.
