# Technical Requirements / Research — US-022

- Hook into internal/parser at the call-argument and composite-literal element
  sites. Nodes VariantLit/LabeledArg/SpreadElement already exist in internal/ast
  (added in US-016).
- A `Name: Expr` argument inside a call distinguishes a VariantLit (labeled-arg
  construction) from an ordinary CallExpr.
- `...expr` inside a composite literal / call becomes ast.SpreadElement.
- Follow no-semicolon structural parsing discipline (carried from US-017..US-021).
