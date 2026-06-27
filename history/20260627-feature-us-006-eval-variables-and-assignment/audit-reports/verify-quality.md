# Verify — Quality (US-006)

## Error handling
- Undefined read and undeclared assign both surface the existing
  `*NotFoundError` (errors.As-detectable), proven by tests — never a silent
  zero/define.
- Unsupported assignment operator, non-ident LHS target, mismatched
  name/value counts, and non-var/const decl-in-statement all return descriptive
  named errors rather than silently no-oping.
- Compound-operator kind mismatches propagate the existing `applyBinary`
  operator/kind-mismatch error.

## Correctness
- RHS values are all evaluated before any binding, so parallel assignment
  (`a, b = b, a`) is correct by construction.
- `Env.Assign` walks the chain and writes in the owning scope (mutation, not
  shadowing); `Define` still binds in the current scope. The two semantics are
  kept distinct, resolving the US-003-deferred question.
- Compound assigns reuse `applyBinary`, so int/float/string operator semantics
  match the expression evaluator exactly.

## No findings
No CRITICAL/MAJOR/MINOR. Change is confined to internal/interp and dependency-
clean (errors/fmt/strconv + goal/internal/{ast,sema,token}).

## Assumptions
- Bitwise/shift compound operators are intentionally unmapped (return a
  descriptive error); they are absent from the current Go-subset corpus and
  documented Out of Scope.
- `var x T` composite zero values remain nil until US-009/US-010.
