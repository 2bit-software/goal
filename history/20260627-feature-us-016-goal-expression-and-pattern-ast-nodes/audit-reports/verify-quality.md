# Verify: Quality — US-016

## Checks
- Node design matches the existing internal/ast conventions (token.Pos fields,
  Pos()/End() with nil-guards mirroring goal_decl.go, unexported category
  markers). VariantLit/VariantPattern guard optional Lparen/Rparen/Binding
  against `token.Pos{}` / nil, consistent with Variant in goal_decl.go.
- Walk cases follow the established helpers: walkExpr for optional Expr children,
  walkExprList for Args, explicit nil-guarded Walk for *Ident and the `Node`
  Body of MatchArm. RestPattern correctly has no children.
- The test asserts what it claims: it does not merely construct nodes — it walks
  them and checks per-child visit counts of exactly 1, and the distinct-type
  assertion is a real string inequality plus exact expected-type checks.
- No error handling is specified for node construction (pure data); Walk's
  nil-node / nil-child no-op behavior is unchanged and already covered by
  TestWalkNilNodeIsNoop.

## Findings
- None CRITICAL / None MAJOR.
- MINOR: parsing and lowering of these nodes are deferred to later stories by
  design (per spec Out of Scope); no consumer exercises them at runtime yet,
  which is expected for a node-definition story.

## Conclusion
Implementation satisfies the spec; all gates green. Ready to complete.
