# Spec — US-022 Parse construction, labeled args, spread

## Behavior
- A call `Enum.Variant(label: value, ...)` with >=1 labeled arg parses to
  *ast.VariantLit (Enum, Variant, Args of *ast.LabeledArg). All-positional calls
  remain *ast.CallExpr.
- A composite-literal element `...X` parses to *ast.SpreadElement (covers
  `...defaults` and `...derive(e)`).

## Acceptance
- Test parses Status.Active(since: now()) -> VariantLit with one LabeledArg.
- Test parses a literal containing ...defaults -> SpreadElement over `defaults`.
