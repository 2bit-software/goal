# Implementation Tasks — US-017 Eval question-mark unwinding

## Task 1: Thread the current-function signature stack
**Status**: completed
**Files**: `internal/interp/interp.go`
**Depends on**: (none)
**Spec coverage**: FR-2, FR-3, FR-4, FR-5 (the enclosing function shape is read
from this stack at propagation time)
**Verify**: `go build ./...`

### Instructions
- Add `fnStack []sema.FuncSig` to the `Interp` struct.
- In `callFunc`, after validating the callee, push the callee's signature
  `ip.fnStack = append(ip.fnStack, ip.sigFor(fn.Name))` and pop with a `defer`
  so any unwind (returnSignal / panicSignal / error) still balances the stack.
- In `callMethod`, push a zero `sema.FuncSig{}` (none-shaped) and pop via defer —
  a `?` inside a method body is therefore the FR-5 refusal (out of scope).
- Add helpers: `sigFor(name string) sema.FuncSig` (reads
  `ip.info.FuncSignatures[name]`, nil-safe -> zero), and `curSig() (sema.FuncSig,
  bool)` (top of `fnStack`; ok=false when empty).
- No behavior change yet; this only records context.

## Task 2: Evaluate postfix `?` (UnwrapExpr) with propagation
**Status**: completed
**Files**: `internal/interp/eval.go`
**Depends on**: Task 1
**Spec coverage**: FR-1..FR-5 + closed-E same-E no-conversion + non-variant refusal
**Verify**: `go build ./...` && `go vet ./...`

### Instructions
- Add `case *ast.UnwrapExpr:` to `evalExpr` -> `return ip.evalUnwrap(e, scope)`.
- `evalUnwrap(u, scope)`:
  - Evaluate `u.X`. (An inner call pushes/pops the callee sig, so on return
    `curSig()` reflects the ENCLOSING function.)
  - If `v.Kind == KindVariant && v.Variant.TypeID == resultTypeID`:
    - `resultOkTag` -> return `payloadValue(v.Variant)` (unwrapped success).
    - `resultErrTag` -> `return Value{}, ip.propagateErr(u, errPayload)`.
  - If `optionTypeID`:
    - `optionSomeTag` -> return `payloadValue` (unwrapped).
    - `optionNoneTag` -> `return Value{}, ip.propagateNone(u)`.
  - Otherwise a located refusal: `interp: %s: cannot use ? on %s (not a Result or
    Option)` using `u.Question` / `u.Pos()`.
- `propagateErr(u, errVal)`:
  - `sig, ok := ip.curSig()`; if `!ok || sig.Mode == ModeNone || sig.Mode ==
    ModeOption` -> located refusal `? in a non-Result function`.
  - `ModeResult` -> `returnSignal{vals: []Value{VariantVal(resultTypeID,
    resultErrTag, map[string]Value{resultErrField: errVal})}}`.
  - `ModeResultClosed` -> `calleeE := ip.calleeErrType(u.X)`; `out := errVal`;
    if `calleeE != "" && calleeE != sig.E`: look up
    `ip.info.FromRegistry[[2]string{calleeE, sig.E}]`; if missing -> located
    refusal naming both types; else look up the conversion func value in the root
    scope by `ConvEntry.Name`, `callFunc` it with `[]Value{errVal}`, take
    `result[0]` as `out`. Then wrap `Result.Err(out)` as above.
- `propagateNone(u)`: `sig, ok := ip.curSig()`; if `!ok || sig.Mode != ModeOption`
  -> located refusal; else `returnSignal{vals: []Value{VariantVal(optionTypeID,
  optionNoneTag, nil)}}`.
- `calleeErrType(x ast.Expr) string`: if `x` is `*ast.CallExpr` with `*ast.Ident`
  Fun, return `ip.info.FuncSignatures[name].E`; else "".
- Reuse existing constants/ctors from value.go; no new imports expected beyond
  `fmt` (already imported).

## Task 3: Unit tests over 05/06 shapes
**Status**: completed
**Files**: `internal/interp/question_test.go` (new)
**Depends on**: Task 2
**Spec coverage**: all acceptance criteria
**Verify**: `go test ./internal/interp/ -run TestQuestion -count=1`

### Instructions
- `package interp`, stdlib `testing` only (NO testify). Reuse `newInterp` and
  `evalFn` / `evalFnErr` (call_test.go / composite_test.go). Write parameterless
  propagating functions so `evalFn(t, ip, "name")` drives them directly.
- Tests:
  - `TestQuestionResultOkContinues` — callee returns `Result.Ok`; caller
    `cfg := parse()?; return Result.Ok(cfg)`; assert returned variant is
    `Result.Ok` with the chained payload.
  - `TestQuestionResultErrEarlyReturns` — callee returns `Result.Err(...)`;
    caller has a post-`?` `return Result.Ok(Config{Raw: "reached"})`; assert the
    result is the propagated `Result.Err` (the "reached" Ok did NOT run).
  - `TestQuestionOptionSomeContinues` / `TestQuestionOptionNoneEarlyReturns`.
  - `TestQuestionClosedEAppliesFromConversion` — `from func toApp(e ParseError)
    AppError`; callee `parse` returns `Result.Err(ParseError.Empty)`; caller
    returns `Result[Config, AppError]`; assert `Err` payload is
    `AppError.Wrapped` with `cause` = the ParseError variant.
  - `TestQuestionClosedESameENoConversion` — same E; assert Err payload is the
    unchanged `ParseError` variant.
  - `TestQuestionOutsidePropagatingFunctionRefused` — `?` in a void function ->
    `evalFnErr`, assert message mentions `?`.
  - `TestQuestionOnNonVariantRefused` — `?` on a plain int -> `evalFnErr`.

## Task 4: Verify gates green, mark story, log progress
**Status**: completed
**Files**: `prd.json`, `progress.txt`
**Depends on**: Task 3
**Spec coverage**: acceptance gate
**Verify**: `go build ./...` && `go vet ./...` && `go test ./... -count=1`
(plus `go list -deps goal/internal/interp | grep -E 'go/types|internal/backend|internal/typecheck'` returns nothing — US-022 envelope)

### Instructions
- Run all three verifyCommands; fix any failure before proceeding.
- Set US-017 `passes: true` in prd.json.
- Append a progress.txt entry and any reusable pattern to the Codebase Patterns
  block.
