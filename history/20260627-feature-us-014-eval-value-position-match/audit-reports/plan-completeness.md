# Plan Audit — Coverage

Every spec requirement traces to a plan element:

- FR-1 (match yields a value) -> `evalMatch` + `evalArmValue` in eval.go.
- FR-2 (three positions) -> `evalExpr` dispatch covers `return`/`:=`/`var =` since
  all three reach `evalExpr` via existing callers (execReturn/execAssign/execDecl).
- FR-3 (uniform dispatch) -> `selectMatchArm` + `armScopeFor` shared by both
  `execMatch` and `evalMatch`.
- FR-4 (loud default) -> `evalMatch` raises `panicSignal` on a nil arm.
- All acceptance criteria map to named tests in `value_match_test.go`.

No scope creep: the refactor of `execMatch`/`execArm` is behaviour-preserving and
exists solely to share dispatch with the new value path.

No CRITICAL/MAJOR findings.

## Assumptions

- Refactoring `execMatch`/`execArm` to use the shared helpers is acceptable (it
  keeps statement-position tests green and avoids duplicated dispatch logic).
