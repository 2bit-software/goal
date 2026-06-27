# Closed-E Result lowering — Business Specification

## Overview

goal's `Result[T, E]` type carries a success value of type T and a failure value
of type E. When E is the built-in `error`, the Result lowers to Go's native
`(T, error)` pair (already handled). When E is a *closed* error type — any type
that is not `error`, such as a user enum — the Result instead lowers to an
explicit sum type (the §8.1 `Ok[T,E]`/`Err[T,E]` encoding). This feature makes
the new AST backend emit that closed-E encoding so closed-E Result programs
transpile to Go that compiles and runs.

## Functional Requirements

### FR-1: Generic Result prelude
When a file contains any function returning a closed-E Result, the backend emits
the generic sum encoding (the Result interface, the Ok and Err structs, and their
marker methods) exactly once in that file, before the declarations that use it.

### FR-2: Sum constructors
Inside a closed-E Result function, `Result.Ok(x)` and `Result.Err(x)` produce the
corresponding sum value carrying x.

### FR-3: Closed-E match
A `match` over a closed-E Result value dispatches on whether the value is an Ok or
an Err, binding the carried value in each arm when the arm uses it, and is
provably exhaustive (an impossible third case panics).

### FR-4: Closed-E `?` propagation
A `?` applied to a closed-E Result call unwraps the success value on Ok and, on
Err, returns early from the enclosing closed-E function carrying the failure.

### FR-5: From-conversion across error types
When the `?` callee's error type differs from the enclosing function's error type,
the declared `from func` conversion between them is invoked on the propagated
failure. A `from func` is itself emitted as an ordinary callable function.

## Acceptance Criteria

- [ ] A file with a closed-E Result function emits the generic Result/Ok/Err
      encoding once, ahead of its first use.
- [ ] `Result.Ok`/`Result.Err` in a closed-E function become the Ok/Err sum
      constructors carrying the argument.
- [ ] A closed-E `match` becomes an exhaustive dispatch over Ok/Err with the
      carried value bound per used arm.
- [ ] A closed-E `?` unwraps on success and propagates the failure on error.
- [ ] When error types differ, the declared conversion is applied to the
      propagated failure; the conversion function itself is emitted and callable.
- [ ] All three feature 06-error-e example inputs pass the behavioral tier
      (generated Go builds and vets cleanly) through the new AST backend.

## User Interactions

Developers write closed-E Result code in goal; the transpiler emits Go. No new CLI
surface. The new engine runs behind the existing `--engine=ast` driver flag.

## Error Handling

A `?` whose callee is not a closed-E Result, or a missing `from func` conversion
across differing error types, is a transpile-time error with a descriptive
message (mirroring the legacy splice engine's wording).

## Out of Scope

- Byte-exact golden parity (US-042 regenerates goldens; this story is judged by
  build + vet).
- `derive func` lowering (US-039) — only `from func` emission is in scope here.
- `...defaults`/`assert` (US-038) and doctest sidecars (US-040).
- Package-mode emission of the prelude once per package (file-mode here; the
  package driver's once-per-package emission is unchanged legacy behavior).

## Open Questions

None — the legacy encoding (internal/pass/closed.go) and the three checked-in
06-error-e goldens fully pin the expected behavior.
