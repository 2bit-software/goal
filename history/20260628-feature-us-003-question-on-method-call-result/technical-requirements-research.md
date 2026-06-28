# Technical Requirements / Research

## Where the gap lives

A `?` site is validated in three layers, all of which currently mis-handle a
`recv.M()` method callee:

- **Lexical checker** (`goal check`): `internal/check/question.go` →
  `internal/analyze`. `analyze.ResolveCallee` (internal/analyze/methods.go)
  already resolves a `recv.Method()` receiver's declared type and looks the
  method up in `Tables.Methods`/`Tables.ForeignMethods`. BUT the resolved
  `analyze.Method` carries only `Arity`/`EndsInError` computed by `countReturns`
  /`endsInError` over the RAW result text — it never detects the `Result[T,
  error]`/`Option[T]` mode the way `analyzeSig` does for a plain function. So a
  `Result[int, error]`-returning method resolves with `Mode==ModeNone`,
  `EndsInError==false`, and `goal check` wrongly emits
  `question-callee-no-error`. Interface-typed receivers are not resolved at all
  (only `Tables.Methods`, not `Tables.Interfaces`).
- **Backend** (`internal/backend`): `emitter.calleeSig` (emit.go) resolves only a
  package-qualified `pkg.Fn(...)` selector; a `recv.M(...)` selector falls
  through to the default two-value destructure, which over-destructures an
  error-only method call (criterion 3).
- **Sema** (`internal/sema`, used by the interp gate): `Method` carries no
  Result/Option mode either, so a method callee cannot be resolved from the
  receiver's method set. Sema does not *wrongly reject* the binding form, but its
  `Method` needs the full signature for the backend to read.

## Plan

1. `internal/analyze`: compute a method's full `FuncSig` (mode/T/E + lowered
   arity/ends-in-error) from its result clause — the same normalization
   `analyzeSig` applies to a plain function — and store it on `analyze.Method`.
   `ResolveCallee` returns that signature, and also resolves interface-typed
   receivers via `Tables.Interfaces`. `methodFrom` is shared by both concrete
   (`analyzeMethods`) and interface (`parseInterfaceBody`) method parsing, so one
   change covers both.
2. `internal/sema`: add the resolved `FuncSig` to `sema.Method`, populated in
   `resolveMethod` (concrete) and `resolveInterface` (interface) via the existing
   `funcSig` helper.
3. `internal/backend`: track the enclosing function's receiver+parameter type map
   in the emitter, and extend `calleeSig` to resolve a `recv.M(...)` selector to
   the method's `sema.FuncSig` so the `?` destructure arity matches.

## Tests

- `internal/check/question_test.go`: a `?` on a `Result[T, error]`-returning
  method is accepted (no error); a `?` on a non-error-ending method is rejected
  with a descriptive diagnostic.
- `internal/backend/backend_test.go`: a value-binding `?` on a Result method binds
  + propagates; a bare `?` on an error-only method lowers to the single-variable
  `if err := recv.M(); err != nil` form.
