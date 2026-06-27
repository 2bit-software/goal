# Technical Requirements / Research — US-017

## Established seams (from progress.txt + internal/interp)

- Non-local control flow already rides the `(… error)` return channel as sentinel
  signal types: `returnSignal{vals}`, `breakSignal`, `continueSignal`,
  `panicSignal{value}` (internal/interp/interp.go). `?` propagation is a
  non-local early return, so it reuses the `returnSignal` mechanism — the
  enclosing `callFunc`/`callMethod` boundary recovers it via `errors.As`.
- Result/Option are universal tagged-union `Variant` values (no (T,error)/*T
  optimization): `resultTypeID`/`resultOkTag`/`resultErrTag`/`resultErrField`,
  `optionTypeID`/`optionSomeTag`/`optionNoneTag` (value.go). `payloadValue` reads
  the single anonymous payload.
- `ast.UnwrapExpr{X, Question}` is the postfix `?` node. It is an `ast.Expr`, so
  the single eval seam is a `case *ast.UnwrapExpr` in `evalExpr` (eval.go). All
  statement positions (`name := expr?`, `_ := expr?`, bare `expr?`) reach it
  through existing callers (execAssign RHS, ExprStmt -> evalExpr).
- The enclosing function's error-propagation shape comes from
  `sema.Info.FuncSignatures[name]` (Mode/T/E). REWRITE: thread the current
  function's `FuncSig` so `?` knows whether to wrap an open-E `Result.Err`, a
  closed-E `Result.Err` (with `from` conversion), or `Option.None`. Implemented
  as an interpreter-held stack pushed in `callFunc`/`callMethod`.
- Closed-E `from` conversion: `sema.Info.FromRegistry[[2]string{calleeE,
  callerE}]` -> `ConvEntry.Name`; the conversion `from func` is an ordinary
  callable registered in the root scope (registerFuncs), so apply it via
  `callFunc`.

## Plan

1. Add a current-function signature stack to `Interp`; push the callee's
   `FuncSig` in `callFunc` (and a zero sig in `callMethod`), pop via defer.
2. Add `case *ast.UnwrapExpr` to `evalExpr` -> `evalUnwrap`.
3. `evalUnwrap`: evaluate operand; Ok/Some -> unwrapped payload; Err -> raise
   `returnSignal` carrying the enclosing function's `Result.Err(...)` (applying
   the `from` conversion when callee E != caller E); None -> raise `returnSignal`
   carrying `Option.None`. A `?` in a non-Result/Option function is a located
   refusal.
4. Tests: `internal/interp/question_test.go` over 05/06 shapes (stdlib testing,
   no testify).

## Constraints

- Zero dependency; stdlib `testing` only (NO testify).
- internal/interp must NOT gain a dependency on go/types, internal/backend, or
  internal/typecheck (US-022 envelope) — reading sema.Info is fine.
- verifyCommands: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
