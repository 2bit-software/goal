# Audit: Completeness — US-016

## Findings

### MINOR — Match arm body type left implicit
The spec says a match arm carries "its pattern and its body" but does not pin
whether a body is an expression (value position) or a statement/block. For a
node-only story this is acceptable: the body is modeled as a generic `Node` so
both forms slot in, mirroring how the existing `MatchArm` doc in the
architecture is open. No blocker.

### MINOR — VariantPattern payload binding cardinality
The observed syntax binds the whole payload to a single identifier
(`Status.Active(a)`). The spec does not enumerate multi-binding patterns. Single
optional binding covers the corpus; later stories can extend if needed.

### None CRITICAL / None MAJOR
All six functional requirements map directly to a named node type and to a Walk
traversal already exemplified by US-015. Acceptance criteria are test-writable.

## Assumptions

- MatchExpr is an *expression* node (implements exprNode), with statement-position
  match represented by using it where a statement is expected, per
  REWRITE-ARCHITECTURE.md lines 158-160.
- MatchArm is a plain support Node (no category marker), like Field/Variant.
- VariantPattern carries a single optional payload `Binding *Ident`.
- LabeledArg and SpreadElement are Expr so they appear in argument / element
  lists.
