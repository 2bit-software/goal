# US-015 Eval Result as tagged union — Business Specification

## Overview

The goscript tree-walking interpreter evaluates the shared AST + sema front-end
directly. Where the Go backend lowers `Result[T, E]` to a Go `(T, error)` tuple,
the interpreter SHALL NOT: it represents `Result` as a universal tagged-union
runtime value, identically to enums and (later) Option. This story makes
`Result.Ok` and `Result.Err` construct tagged-union values and makes `match`
consume them by tag, binding the unwrapped payload.

## Functional Requirements

### FR-1: Construct Result.Ok and Result.Err
The interpreter SHALL evaluate `Result.Ok(x)` to a tagged-union value carrying the
`Ok` tag and the value `x`, and `Result.Err(e)` to a tagged-union value carrying
the `Err` tag and the error `e`.

### FR-2: Uniform representation for open-E and closed-E
The interpreter SHALL represent Result identically whether `E` is `error` (open-E,
e.g. `Result[Config, error]`) or an enum (closed-E, e.g. `Result[Config,
ParseError]`). There SHALL be no `(T, error)` optimization and no special encoding
that distinguishes the two at runtime.

### FR-3: Match over Result
The interpreter SHALL dispatch a `match` over a Result value on its tag: an
`Result.Ok(name)` arm runs when the value is `Ok` and binds `name` to the
unwrapped payload value; an `Result.Err(name)` arm runs when the value is `Err`
and binds `name` to the unwrapped error value.

## Acceptance Criteria

- [ ] `Result.Ok(x)` evaluates to a tagged-union value whose tag is `Ok` and whose
      payload is `x`.
- [ ] `Result.Err(e)` evaluates to a tagged-union value whose tag is `Err` and
      whose payload is `e`.
- [ ] A `match` over an `Ok` value runs the `Ok` arm and binds the unwrapped
      payload; over an `Err` value runs the `Err` arm and binds the unwrapped
      error.
- [ ] The above hold for an open-E shape (03-result: `Result[Config, error]`,
      Err carries a host `error`) and a closed-E shape (06-error-e:
      `Result[Config, ParseError]`, Err carries an enum variant), with the same
      runtime representation.
- [ ] An unknown `Result.<X>` constructor or a wrong argument count is a located,
      descriptive error, never a silent value.

## User Interactions

None directly. This is interpreter-internal behavior exercised by goal programs
that use `Result` and observed through unit tests.

## Error Handling

- An unknown `Result` constructor name (anything other than `Ok`/`Err`) yields a
  descriptive refusal naming the bad constructor.
- A `Result.Ok`/`Result.Err` call with other than exactly one argument yields a
  descriptive refusal.
- A `match` whose Result tag matches no arm (impossible in a sema-exhaustive
  program) raises the existing loud `unreachable` panic, never a silent
  fall-through.

## Out of Scope

- Option (`Some`/`None`) — US-016.
- Postfix `?` early-return unwinding over Result/Option — US-017.
- `from`-conversion application during `?` propagation for closed-E — US-017.
- Any change to the Go backend's `(T, error)` lowering.

## Open Questions

None.
