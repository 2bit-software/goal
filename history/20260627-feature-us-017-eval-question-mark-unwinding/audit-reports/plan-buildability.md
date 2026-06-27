# Plan Buildability Audit — US-017

## Checks

- **Dependency order valid.** fnStack field (1) -> evalUnwrap/propagation (2,3)
  -> tests (4). No forward references.
- **Interface contracts agree.** `returnSignal{vals []Value}` already exists and
  is recovered by `callFunc`/`callMethod` via `errors.As`; `VariantVal`,
  `payloadValue`, `resultTypeID`/`optionTypeID` constants exist in value.go;
  `sema.FuncSig`/`Mode`/`FromRegistry`/`ConvEntry` exist in sema. `evalExpr`
  signature `(ast.Expr, *Env) (Value, error)` accommodates the new case.
- **File paths verified.** interp.go, eval.go, value.go exist; question_test.go
  is new and non-conflicting.
- **Compiles at each step.** Adding the fnStack field + push/pop is
  self-contained; the UnwrapExpr case + helpers only reference existing symbols.
- **Integration points specific.** evalExpr case + execAssign RHS / ExprStmt
  default path named; callFunc/callMethod recovery path named; from-func root-scope
  callable named.
- **Test harness verified.** `newInterp` (call_test.go) parses + `sema.Resolve` +
  `New`; `evalFn` (composite_test.go) calls a named function via evalExpr and
  asserts the returned Value. A `?`-raised `returnSignal` is recovered by callFunc
  so `evalFn` sees the propagated Result/Option variant — confirmed by reading the
  helpers.

## Findings

None CRITICAL/MAJOR. Buildable as ordered.

## Assumptions

- `curSig()` reads the top of fnStack; an empty stack (e.g. a hand-built
  evalUnwrap call with no enclosing callFunc) is treated as the FR-5 refusal.
