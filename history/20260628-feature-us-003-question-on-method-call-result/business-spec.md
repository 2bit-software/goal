# Business Spec — `?` on method calls returning Result

## Outcome

A goal author can write `?` directly on a method call whose method returns a
`Result` (or a trailing `error`), without first wrapping the call in a plain
helper function.

## Acceptance criteria

1. `goal check` accepts `v := recv.M()?` where `M` returns `Result[T, error]`,
   emitting no `question-callee-no-error` diagnostic.
2. Transpiling `v := recv.M()?` (M returning `Result[T, error]`) yields valid Go
   that binds the value and propagates the trailing error.
3. A bare `recv.M()?` whose method returns only `error` (no value) lowers to the
   single-variable `if err := recv.M(); err != nil` form, not a two-value
   destructure.
4. `?` on a method whose return does not end in an error is still rejected with a
   descriptive diagnostic.
5. A checker test and a backend test covering `?` on a method-call callee
   returning a `Result` exist and pass.

## Constraints

- No behavior change for existing plain-function and package-qualified `?`
  callees, nor for the error-only stdlib `?` lowering.
- Both concrete-struct and interface-typed method receivers are in scope (the
  motivating case is interface calls, which today are wrapped in helper funcs).
