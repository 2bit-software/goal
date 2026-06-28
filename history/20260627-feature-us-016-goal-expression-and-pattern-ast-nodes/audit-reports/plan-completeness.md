# Plan Audit: Coverage — US-016

## Findings

### None CRITICAL / None MAJOR

Every spec requirement traces to a plan element:
- FR-1 (match expr/arms) -> MatchExpr, MatchArm.
- FR-2 (destructure/rest) -> VariantPattern, RestPattern.
- FR-3 (postfix ?) -> UnwrapExpr.
- FR-4 (construction + labeled arg) -> VariantLit, LabeledArg.
- FR-5 (spread) -> SpreadElement.
- FR-6 (traversal) -> walk.go cases + TestWalkGoalExprChildren.

Acceptance criteria all have a testing strategy: node existence (compile),
distinct-type assertion (%T inequality), per-child Walk visit count, and the
three prd verify gates.

No scope creep — the plan adds exactly the eight named nodes and their Walk
cases, nothing adjacent (no parsing, no lowering).

### MINOR — Body type for MatchArm is `Node`
Modeling MatchArm.Body as `Node` (not `Stmt` or `Expr`) is deliberate so both
value-position and statement-position bodies fit; flagged only so it is a
conscious choice. Not a blocker.

## Assumptions

- MatchExpr is an Expr; statement-position use is via context (no separate
  MatchStmt), per architecture lines 158-160.
- Single optional payload Binding on VariantPattern.
