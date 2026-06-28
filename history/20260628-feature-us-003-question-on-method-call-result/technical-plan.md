# Technical Plan — US-003 `?` on method calls returning Result

## Layer 1 — lexical checker (`internal/analyze`)

- `internal/analyze/analyze.go`
  - Add `Return FuncSig` to `type Method`.
  - New helper `methodResultSig(results string) FuncSig`: detect a `Result[T, E]`
    / `Option[T]` result clause (string form) and apply the same lowered
    arity/ends-in-error normalization `analyzeSig` uses; otherwise fall back to
    `countReturns`/`endsInError`.
  - In `methodFrom`, set `Return: methodResultSig(results)` and derive the
    Method's `Arity`/`EndsInError` from it (consistency). This covers both
    concrete methods (`analyzeMethods`) and interface methods
    (`parseInterfaceBody`), which both build through `methodFrom`.
- `internal/analyze/methods.go`
  - `ResolveCallee`: for an in-file concrete method, return `m.Return`. Also
    resolve interface-typed receivers via `t.Interfaces[base]`.

## Layer 2 — sema (`internal/sema`)

- `internal/sema/sema.go`: add `Return FuncSig` to `type Method`.
- `internal/sema/resolve.go`: `resolveMethod` sets `Return: funcSig(d.Name.Name,
  d.Type)`; `resolveInterface` sets `Return: funcSig(n.Name, ft)`. Reuses the
  existing `funcSig` helper, which already detects Result/Option mode.

## Layer 3 — backend (`internal/backend`)

- `internal/backend/emit.go`
  - Add `recvTypes map[string]string` to `emitter` (ident -> base type name).
  - In `funcDecl`, build it from `d.Recv` + `d.Type.Params`, save/restore around
    the body like the other per-function state.
  - Extend `calleeSig`'s `*ast.SelectorExpr` case: when `fn.X` is an identifier
    bound in `recvTypes`, resolve the method's `sema.FuncSig` from
    `info.Methods[type]` then `info.Interfaces[type]` and return it; otherwise
    keep the existing package-qualified path.

## Tests

- `internal/check/question_test.go`: Result-returning method `?` accepted;
  non-error-ending method `?` rejected with `question-callee-no-error`.
- `internal/backend/backend_test.go`: value-binding `?` on a Result method binds +
  propagates; bare `?` on an error-only method lowers to the single-variable
  `if err := recv.M(); err != nil` form (and never `_, err := recv.M()`).

## Risk / regression surface

- Existing plain-function and package-qualified `?` paths are untouched (new
  branches only fire for a receiver-bound selector).
- `Method.Arity`/`EndsInError` are read only by `ResolveCallee`; deriving them
  from the same `FuncSig` keeps existing readers correct.
- Full `verifyCommands` (build/vet/test) gate the change.
