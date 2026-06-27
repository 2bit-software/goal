# Research — US-017 Eval question-mark unwinding

This story is fully determined by established internal seams (US-012..US-016);
no external/web research applies. Findings are codebase-internal.

## How prior runtime mechanics were threaded (precedent)

- `match` (US-013/014) is dispatched on `Variant.Tag` and binds payload via
  `armScopeFor`; Result/Option were bound UNWRAPPED keyed on `Variant.TypeID`.
  `?` reuses the same tagged-union shape — no new value model.
- Non-local control flow uses sentinel error types recovered at boundaries
  (`returnSignal` at the call boundary; `break`/`continue` at loops;
  `panicSignal` only at the host). `?` early-return is exactly a `returnSignal`
  raised mid-expression and recovered by the enclosing `callFunc`.

## Three approaches considered for "what does `?` early-return"

### Option: dedicated `questionSignal` recovered + re-wrapped in callFunc
- Pros: keeps wrapping logic in one place.
- Cons: callFunc would need the enclosing FuncSig anyway, and would have to
  distinguish question-unwind from ordinary return; more moving parts.

### Option: `?` raises a plain `returnSignal` carrying the already-built Err/None
- What it is: `evalUnwrap` reads the enclosing FuncSig, constructs the
  `Result.Err(...)`/`Option.None` variant itself, and raises `returnSignal{vals}`.
- Pros: zero changes to callFunc's recovery; the value is identical in shape to
  what an explicit `return Result.Err(...)` produces, so the boundary needs no
  special case. Matches the AST backend's "? returns the function's own
  Err/None" lowering (progress.txt).
- Cons: `evalUnwrap` needs access to the current FuncSig — solved with a stack.
- **Chosen.**

### Option: encode `?` as a try/recover over Go panic
- Cons: inconsistent with the project's explicit-signal discipline (panicSignal
  is reserved for `panic`, never recovered except at the host). Rejected.

## Closed-E `from` conversion

`sema.Info.FromRegistry[[2]string{calleeE, callerE}]` gives `ConvEntry.Name`; the
`from func` is an ordinary callable already in the root scope (registerFuncs),
applied via `callFunc` to the propagated error before re-wrapping `Result.Err`.
Callee E type comes from `FuncSignatures[calleeName].E` (calleeName read off the
`UnwrapExpr` operand when it is a direct call). Same-E (or unresolvable callee)
needs no conversion.

## Fixtures (oracle)

- features/05-question-prop/examples: qprop_result (Result Ok chain),
  qprop_option (Option Some/None chain), qprop_bare/_discard/_erronly.
- features/06-error-e/examples: qclosed_prop_same (same E, no conversion),
  qclosed_prop_from (`from func toApp` ParseError -> AppError conversion).

## Audit resolution (iteration 2): test harness + branch coverage

Audit found the named fixtures return `Ok`/`None` UNCONDITIONALLY, so loading
them verbatim never reaches the Err/None/`from`-conversion branches. Resolution,
consistent with the existing interp test convention (result_test.go,
option_test.go use INLINE `const program` strings + `newInterp`/`evalFn`, NOT
`.goal` file loading; the `.go.expected` files are backend Go output, not an
interpreter oracle):

- Tests use inline programs MODELED ON the 05/06 shapes but adapted so each
  branch actually fires:
  - Ok-continue: a `parse`/`readFile` that returns `Result.Ok(...)`, chained via
    `?`, asserting the unwrapped value flows through.
  - Err-early-return: a `parse` that returns `Result.Err(...)` for a sentinel
    input (mirrors qclosed_prop_same's `if s == "" { return Result.Err(...) }`),
    asserting the caller short-circuits with that Err and later statements do not
    run.
  - Some-continue / None-early-return: an Option function returning Some vs None
    by input, asserting unwrap vs early `Option.None`.
  - Closed-E `from`: a program with `from func toApp(e ParseError) AppError`
    where the callee `parse` DOES return `Result.Err(ParseError.Empty)`, asserting
    the caller (returning `Result[Config, AppError]`) returns
    `Result.Err(AppError.Wrapped(cause: ...))` — the conversion fired.
  - Same-E: callee and caller share E (ParseError); assert no conversion (the Err
    payload is the unchanged ParseError variant).
- Out of asserted scope: `?` inside a method, and `?` on a bare `func(...) error`
  callee (the qprop_erronly shape) — handled best-effort only.
- Empty/none-shaped enclosing sig (e.g. `?` in a void/`main` function) is the
  FR-5 located refusal.
- The conversion trigger is resolved STATICALLY off the direct-call operand's
  callee `FuncSignatures[name].E` vs the enclosing caller `E`. A closed-E
  propagation whose required `from` conversion cannot be resolved is a located
  refusal, never a silent mistyped Err.
