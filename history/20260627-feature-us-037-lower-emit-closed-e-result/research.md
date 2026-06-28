# Research — US-037 (internal codebase analysis)

No external research required: this is a faithful reimplementation of a known-good
encoding (internal/pass/closed.go) onto the AST backend, reading sema.Info facts
instead of token scans. Summary of findings:

## Closed-E semantics (from internal/pass/closed.go + features/06-error-e goldens)

- A closed-E Result is `Result[T, E]` where E is NOT `error`. sema.Resolve marks
  it `Mode == ModeResultClosed` with `T`/`E` strings on the FuncSig.
- The signature stays as written (`Result[Config, ParseError]`) — it is satisfied
  by the injected generic prelude. resultOptionKind already returns roNone for
  closed-E, so funcSig emits the type verbatim (no change needed there).
- Prelude (emitted once when any func is closed-E):
  `type Result[T,E any] interface{ isResult() }` / `Ok[T,E]{Value T}` /
  `Err[T,E]{Value E}` + two marker methods.
- ctor: `Result.Ok(X)` -> `Ok[T,E]{Value: X}`, `Result.Err(X)` -> `Err[T,E]{Value: X}`.
- match: type-switch on `Ok[T,E]`/`Err[T,E]`, `binding := g.Value` alias per used
  arm, panicking default. Dispatched on the SCRUTINEE callee's mode
  (ModeResultClosed), not the enclosing function — `handle` is void.
- `?`: `var name T` then `switch g := callee().(type) { case Ok[T,E]: name = g.Value;
  case Err[T,E]: return Err[callerT,callerE]{Value: errValue}; default: panic }`.
  errValue is `g.Value` when callee.E == caller.E, else `conv(g.Value)` where conv
  is `FromRegistry[[2]string{callee.E, caller.E}].Name`.
- `from func` is registered in FromRegistry (NOT FuncSignatures); it must emit as
  an ordinary Go function (modifier stripped) so its body (already lowerable, e.g.
  a VariantLit construction) compiles and is callable.

## Confidence: High

The three goldens (qclosed_match, qclosed_prop_same, qclosed_prop_from) confirm
every shape, and the legacy pass is the source of truth. The AST backend already
has the seams (fnKind, gensym, calleeSig/calleeMode, armBody) US-034/035/036 added.
