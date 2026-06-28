# Plan Audit — Coverage (US-006)

Every FR maps to a plan element:
- FR-1 var decl → execStmt DeclStmt VAR (with zeroValue for no-initializer).
- FR-2 `:=` → execStmt AssignStmt DEFINE.
- FR-3 const → execStmt DeclStmt CONST.
- FR-4 plain `=` (incl. parallel) → AssignStmt ASSIGN + Env.Assign + RHS-first.
- FR-5 compound → compoundBinOp + applyBinary + Env.Assign.
- FR-6 reads/errors → Ident Lookup + NotFoundError on undeclared assign.

Each acceptance criterion has a named test. No scope creep: plan touches only
the three interp files. No CRITICAL/MAJOR findings.

## Assumptions
- A non-ident assignment target (index/selector) is out of scope here and
  returns a descriptive error rather than being silently handled.
