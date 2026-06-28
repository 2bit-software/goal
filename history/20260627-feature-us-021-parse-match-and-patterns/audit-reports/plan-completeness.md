# Plan Audit — Coverage

Every spec requirement maps to a plan element:

- FR-1 (statement position) → `parseStmt` dispatch hook + status_match test.
- FR-2 (value position) → `parseOperand` hook + `startsExpr` + status_var/return tests.
- FR-3 (variant patterns + binding) → `parseVariantPattern` + binding assertion.
- FR-4 (rest pattern) → `parsePattern` `_` branch + status_rest test.
- FR-5 (arm bodies) → `parseMatchArm` block/expr branch.

No scope creep: every plan element traces to a requirement. No CRITICAL/MAJOR.

## Assumptions

- Tests read corpus inputs from `../../features/...` (cwd = internal/parser),
  matching the existing `readExample` helper in `goal_decl_test.go`.
