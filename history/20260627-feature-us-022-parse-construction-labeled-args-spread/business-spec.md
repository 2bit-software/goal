# Business Spec — US-022 Parse construction, labeled args, spread

As a compiler engineer, I need construction parsing so `Enum.Variant(field: v)`
and `...defaults` become AST nodes.

## Acceptance Criteria
- Parser parses VariantLit with LabeledArg arguments and SpreadElement
  (`...defaults`, `...derive(s)`) inside composite literals.
- A test parses `Status.Active(since: now())` and a literal containing
  `...defaults` and asserts the nodes.
