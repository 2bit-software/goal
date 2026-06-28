# Implementation Tasks

## Task 1: Closed-E facts in lower.go
**Status**: completed
**Files**: internal/backend/lower.go
**Depends on**: (none)
**Spec coverage**: FR-1 (prelude), foundation for FR-2..FR-5
**Verify**: `go build ./internal/backend/...`

### Instructions
- Add `roResultClosed` to the `roKind` enum (after `roOption`).
- Add the `resultPrelude` string constant mirroring internal/pass.ResultPreamble
  (the generic `Result[T,E]` interface, `Ok[T,E]`/`Err[T,E]` structs, two marker
  methods), token-correct Go text.
- Add `needsResultPrelude(info *sema.Info) bool` — true when any
  info.FuncSignatures entry has Mode == sema.ModeResultClosed (nil-safe).

## Task 2: Prelude emission + funcDecl closed-E setup + FuncFrom passthrough
**Status**: completed
**Files**: internal/backend/emit.go
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-5 (from func emission), enables FR-2..FR-4
**Verify**: `go build ./internal/backend/...`

### Instructions
- Add `closedT, closedE string` fields to the emitter.
- In `file()`, emit `resultPrelude` exactly once before the first non-import decl
  (guard `needsResultPrelude(e.info)`; detect import decls via
  `gd.Tok.String() == "import"`). Emit it even if the file has only imports/none.
- In `funcDecl()`, after `kind, _ := resultOptionKind(d.Type)`, look up
  `e.info.FuncSignatures[d.Name.Name]`; if Mode==ModeResultClosed set
  kind=roResultClosed and closedT/closedE = sig.T/sig.E. Save+restore closedT/
  closedE alongside the existing fnKind/okName/errName/taken block.
- Relax the modifier guard to allow `ast.FuncFrom` (emit as an ordinary func, the
  `from` marker has no syntactic residue); keep `ast.FuncDerive` failing (US-039).

## Task 3: Closed ctors, closed `?`, closed match lowering
**Status**: completed
**Files**: internal/backend/emit.go
**Depends on**: Task 2
**Spec coverage**: FR-2 (ctors), FR-3 (match), FR-4 (`?`), FR-5 (From-conversion)
**Verify**: `go build ./internal/backend/... && go vet ./internal/backend/...`

### Instructions
- `returnStmt`: add `case roResultClosed:` calling `emitClosedResultReturn(s.Results[0])`.
  `emitClosedResultReturn` matches `Result.Ok(X)`/`Result.Err(X)` and emits
  `return Ok[closedT, closedE]{Value: X}` / `return Err[...]{Value: X}` (X via exprList).
- `unwrap`: add `case roResultClosed:` -> `unwrapClosed(name, u, discard)`.
  `unwrapClosed`: read callee sig via `calleeSig(u.X)` (require ModeResultClosed,
  else fail). Mint guard via gensym. errValue = `guard+".Value"`, or
  `conv.Name+"("+guard+".Value)"` when `sig.E != e.closedE` using
  `e.info.FromRegistry[[2]string{sig.E, e.closedE}]` (fail if absent). Emit
  `var name sig.T`, then `switch guard := <X>.(type) { case Ok[sig.T,sig.E]:
  name = guard.Value; case Err[sig.T,sig.E]: return Err[closedT,closedE]{Value:
  errValue}; default: panic(<msg>) }`.
- `resultMatch`: replace the closed-E `fail` with a route to `closedResultMatch`
  when `e.calleeMode(m.Subject) == sema.ModeResultClosed`.
  `closedResultMatch`: read callee sig (T/E) via calleeSig; find Ok/Err arms;
  per arm declare `binding := guard.Value` only when the arm uses its binding
  (usesIdent); introduce the guard only when either arm uses its binding; emit the
  Ok/Err type-switch cases + panicking default; emit arm bodies via armBody.

## Task 4: Tests
**Status**: completed
**Files**: internal/backend/backend_test.go
**Depends on**: Task 3
**Spec coverage**: all ACs (behavioral tier + encoding)
**Verify**: `go test ./internal/backend/... -count=1`

### Instructions
- Add `errorEClosedCases` = the 3 features/06-error-e/examples/*.goal inputs.
- `TestASTEngineClosedResultBehavioralTier`: each input through corpus.RunCompile
  via corpus.TranspilerFunc(backend.Transpile), -short-skipped, t.Fatal on empty.
- `TestASTEngineClosedResultEncoding`: assert the prelude, the Ok/Err sum ctors,
  the closed match type-switch cases, the closed `?` shape, and the From-conversion
  (`toApp(`) appear in the emitted Go for the relevant inputs.

## Final verification (all tasks)
`go build ./...`, `go vet ./...`, `go test ./... -count=1` — all green.
