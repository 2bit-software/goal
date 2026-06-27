# Plan — US-022
1. internal/parser/goal_construct.go (new): parseCallArg, makeVariantLit,
   parseSpreadElement.
2. parser.go: parseCallSuffix detects labeled args -> VariantLit; parseElement
   handles leading ELLIPSIS -> SpreadElement.
3. internal/parser/goal_construct_test.go (new): VariantLit + LabeledArg from
   status.goal; positional call stays CallExpr; SpreadElement from
   defaults_primitives.goal; ...derive(e) spread call.
