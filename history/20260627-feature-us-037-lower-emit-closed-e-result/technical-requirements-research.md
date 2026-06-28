# Technical Requirements / Research — US-037

## Reference implementation

internal/pass/closed.go is the known-good splice encoding to mirror:

- `ResultPreamble` — the generic sum encoding (`Result[T,E] interface`,
  `Ok[T,E]`, `Err[T,E]`, marker methods), injected once after package/imports
  when any function returns a closed-E Result (`NeedsResultPrelude`).
- closed ctors: `Result.Ok(X)`/`Result.Err(X)` in a closed-E function ->
  `Ok[T,E]{Value: X}` / `Err[T,E]{Value: X}`.
- closed match: type-switch on `Ok[T,E]`/`Err[T,E]` with a `binding := g.Value`
  alias per used arm and a panicking default.
- closed `?`: `var name T` + type-switch returning `Err[callerT,callerE]{Value:
  errValue}` in the Err arm, where errValue is `g.Value` (same E) or
  `conv(g.Value)` (From-conversion when callee.E != caller.E).

## Facts read off sema.Info (not token scans)

- `FuncSignatures[name]`: Mode (ModeResultClosed), T, E.
- `FromRegistry[[2]string{callee.E, caller.E}]`: ConvEntry{Name}.
- `from func` is routed to FromRegistry by sema.Resolve (not FuncSignatures); it
  must emit as an ordinary Go function (modifier stripped) so the conversion is
  callable.

## Backend seam (internal/backend)

- emit.go: add roResultClosed handling — funcDecl sets fnKind + closedT/closedE
  from the sema sig; returnStmt lowers closed ctors; unwrap dispatches
  roResultClosed -> unwrapClosed; resultMatch routes ModeResultClosed ->
  closedResultMatch; funcDecl allows ast.FuncFrom (emit as plain).
- file(): emit the Result prelude once, after import decls, before other decls,
  guarded by needsResultPrelude(info).
- lower.go: add the prelude constant + needsResultPrelude helper + roResultClosed
  const.

## Test (backend_test.go, external package backend_test)

- errorEClosedCases = the 3 features/06-error-e/examples/*.goal inputs.
- TestASTEngineClosedResultBehavioralTier: each through backend.Transpile +
  corpus.RunCompile (build+vet), -short-skipped.
- TestASTEngineClosedResultEncoding: pin prelude + Ok/Err sum + closed `?`
  type-switch + From-conversion shapes.
