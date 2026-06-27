# Research — US-022

The AST nodes (VariantLit, LabeledArg, SpreadElement) already exist (US-016).
The parser hook points are established by prior stories:
- Call arguments are parsed in parseCallSuffix; a `Label: Value` arg (IDENT
  followed by COLON) marks variant construction -> VariantLit; all-positional
  stays CallExpr.
- Composite-literal elements are parsed in parseElement; a leading ELLIPSIS
  (`...X`) -> SpreadElement. This is goal's PREFIX spread, distinct from Go's
  trailing variadic call spread.
Confidence: High. No external deps; pattern mirrors goal_match.go.
