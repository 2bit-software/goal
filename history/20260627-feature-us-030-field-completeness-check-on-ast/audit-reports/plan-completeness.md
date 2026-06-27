# Plan Audit — Coverage

Every spec requirement maps to a plan element:

- FR-1 struct completeness → CheckFields CompositeLit branch (Error).
- FR-2 variant completeness → CheckFields VariantLit branch (Error).
- FR-3 spread opt-out → SpreadElement short-circuit in CompositeLit branch.
- FR-4 deferral → unknown-Ident keyed-literal Warning.
- FR-5 match-binding ignored → only VariantLit walked, not VariantPattern.
- FR-6 message parity → message templates quoted verbatim from legacy.
- All ACs → corpus runner test over testdata/check/08-no-zero-value.

No scope creep: the plan touches only sema (fields.go + check.go wiring) and a
new corpus test. No CRITICAL/MAJOR findings.

## Assumptions

- A `*ast.KeyValueExpr` key in a struct literal is an `*ast.Ident` (the field
  name) — true for goal struct literals.
