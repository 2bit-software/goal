# Implementation Plan — US-017 Eval question-mark unwinding

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/question_test.go` | Unit tests over 05/06-modeled inline programs: Ok/Some continue, Err/None early return, closed-E `from` conversion + same-E no-conversion, and refusals. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/interp/interp.go` | Add a current-function `FuncSig` stack to `Interp`; push the callee sig in `callFunc` and a none sig in `callMethod`, pop via defer. Add `propagate` helpers building the enclosing Err/None `returnSignal`. |
| `internal/interp/eval.go` | Add `case *ast.UnwrapExpr` to `evalExpr` -> `evalUnwrap` (Ok/Some unwrap; Err/None raise the propagation signal with `from` conversion for closed-E). |

## Package Structure

```
internal/interp/
  interp.go         (modified: fnStack, push/pop, propagation helpers)
  eval.go           (modified: UnwrapExpr case + evalUnwrap + calleeErrType)
  question_test.go  (new)
```

No new packages; no new imports beyond what interp.go/eval.go already use
(`fmt`, `ast`, `sema`). The US-022 dependency envelope is preserved (no go/types,
internal/backend, internal/typecheck).

## Dependency Graph

1. `Interp.fnStack` field + push/pop in `callFunc`/`callMethod` (interp.go).
2. `evalUnwrap` + `case *ast.UnwrapExpr` in `evalExpr` (eval.go), depends on 1.
3. Propagation helpers (`propagateErr`, `propagateNone`, `calleeErrType`),
   depend on 1 + the existing Result/Option Variant constructors.
4. `question_test.go`, depends on 2 + 3.

## Interface Contracts

```go
// interp.go
type Interp struct {
    // ...existing fields...
    fnStack []sema.FuncSig // current-function signature stack (innermost last)
}

// pushed in callFunc:  ip.fnStack = append(ip.fnStack, ip.sigFor(fn.Name))
//                      defer func(){ ip.fnStack = ip.fnStack[:len(ip.fnStack)-1] }()
// pushed in callMethod: ip.fnStack = append(ip.fnStack, sema.FuncSig{}) // none sig
func (ip *Interp) sigFor(name string) sema.FuncSig   // FuncSignatures[name], zero if absent
func (ip *Interp) curSig() (sema.FuncSig, bool)       // top of fnStack

// eval.go
func (ip *Interp) evalUnwrap(u *ast.UnwrapExpr, scope *Env) (Value, error)
func (ip *Interp) propagateErr(u *ast.UnwrapExpr, errVal Value) error   // returns returnSignal or located error
func (ip *Interp) propagateNone(u *ast.UnwrapExpr) error                // returns returnSignal or located error
func (ip *Interp) calleeErrType(x ast.Expr) string                     // FuncSignatures[calleeName].E or ""
```

### evalUnwrap semantics

- Evaluate `u.X`.
- Result Ok -> `payloadValue` (unwrapped success), continue.
- Result Err -> `propagateErr(u, errPayload)`.
- Option Some -> `payloadValue`, continue.
- Option None -> `propagateNone(u)`.
- Non-variant operand -> located refusal.

### propagateErr

- `sig, ok := curSig()`; not ok / `sig.Mode == ModeNone`/`ModeOption` -> located refusal.
- `ModeResult` (open-E): `returnSignal{vals: [Result.Err(errVal)]}`.
- `ModeResultClosed`: `calleeE := calleeErrType(u.X)`; if `calleeE != "" && calleeE != sig.E`,
  look up `info.FromRegistry[[2]string{calleeE, sig.E}]`; missing -> located refusal;
  else call the conversion func (root-scope callable) with `errVal`, take result[0];
  `returnSignal{vals: [Result.Err(converted)]}`.

### propagateNone

- `sig, ok := curSig()`; `sig.Mode != ModeOption` -> located refusal.
- `returnSignal{vals: [Option.None]}`.

## Integration Points

- `evalExpr` (eval.go) already dispatches all expression nodes; adding the
  `*ast.UnwrapExpr` case is the single seam. Statement positions reach it via:
  `cfg := parse(a)?` (execAssign RHS -> evalExpr), `flush()?` (execStmt ExprStmt
  default -> evalExpr), `_ := flush()?` (assignTarget Ident `_`).
- `callFunc`/`callMethod` (interp.go) already recover `returnSignal` via
  `errors.As` and return its `vals` — so a `?`-raised `returnSignal` is returned
  exactly like an explicit `return Result.Err(...)`. No boundary change needed.
- The `from func` conversion is registered as a normal callable in the root scope
  by `registerFuncs`, so it is invoked through `callFunc`.

## Testing Strategy

`internal/interp/question_test.go` (stdlib `testing`, NO testify), using the
existing `evalFn`/`newInterp` helpers (composite_test.go / call_test.go):

- `TestQuestionResultOkContinues` — `?` on Ok yields unwrapped value chained.
- `TestQuestionResultErrEarlyReturns` — `?` on Err returns the caller's
  `Result.Err`; a statement after the `?` (which would change the result) does
  not run.
- `TestQuestionOptionSomeContinues` / `TestQuestionOptionNoneEarlyReturns`.
- `TestQuestionClosedEAppliesFromConversion` — callee errs `ParseError.Empty`,
  caller returns `Result[Config, AppError]`; assert `Err(AppError.Wrapped{cause})`.
- `TestQuestionClosedESameENoConversion` — same E; Err payload is the unchanged
  `ParseError` variant.
- `TestQuestionOutsidePropagatingFunctionRefused` — `?` in a void function ->
  located refusal.
- `TestQuestionOnNonVariantRefused` — `?` on a plain int -> located refusal.

## Risks / Mitigations

- Stack imbalance on panic-unwind: push/pop via `defer` in callFunc/callMethod so
  any unwinding (returnSignal, panicSignal, real error) still pops.
- `evalUnwrap` evaluating `u.X` triggers an inner call that pushes/pops the
  callee's sig before `evalUnwrap` inspects the result, so `curSig()` correctly
  reflects the ENCLOSING function at propagation time.
