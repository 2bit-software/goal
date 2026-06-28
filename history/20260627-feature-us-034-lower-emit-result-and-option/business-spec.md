# US-034 Lower and emit Result and Option — Business Specification

## Overview

The AST (`--engine=ast`) backend currently lowers enums, sealed interfaces, and
the `implements` clause (US-033), but not goal's two core error/optionality
types. This feature adds lowering for **open-E Result** (`Result[T, error]`) and
**Option** (`Option[T]`) so programs using them transpile to native Go that
behaves identically to the legacy splice engine. After this change the
`features/03-result` and `features/04-option` examples transpile through the new
backend and pass the behavioral conformance tier (the generated Go builds and
vets).

## Functional Requirements

### FR-1: Open-E Result return type lowers to a native `(T, error)` pair
A function declared to return `Result[T, error]` SHALL be emitted with a Go
return of two values whose second is `error`, so callers receive an ordinary Go
`(T, error)` result.

### FR-2: Result constructors in return position lower to the pair
Within an open-E Result function, `return Result.Ok(X)` SHALL produce the success
value paired with a nil error, and `return Result.Err(X)` SHALL produce the
zero success value paired with the error X.

### FR-3: Statement-position match over a Result splits on the error
A statement-position `match` whose arms are `Result.Ok` / `Result.Err` SHALL be
emitted as: bind the scrutinee to a value/error pair, then branch — the
`Result.Err` arm body runs when the error is non-nil, the `Result.Ok` arm body
otherwise. The Ok payload binding is available in the Ok arm; the Err payload
binding (the error) is available in the Err arm.

### FR-4: Option type lowers to a pointer
A `Option[T]` type SHALL be emitted as `*T`.

### FR-5: Option constructors in return position lower to the pointer form
Within an Option function, `return Option.None` SHALL produce `nil`, and
`return Option.Some(x)` SHALL produce the address of the value x.

### FR-6: Statement-position match over an Option splits on nil
A statement-position `match` whose arms are `Option.Some` / `Option.None` SHALL
be emitted as: bind the scrutinee pointer, then branch — the `Option.Some` arm
body runs when the pointer is non-nil (with the payload dereferenced and bound),
the `Option.None` arm body otherwise.

## Acceptance Criteria

- [ ] A function returning `Result[int, error]` transpiles to a Go function whose
      return is a `(T, error)` pair, and its body's `Result.Ok`/`Result.Err`
      returns produce the success/error pairs.
- [ ] A statement-position Result match transpiles to an `if err != nil` / `else`
      split that builds and vets.
- [ ] A function returning `Option[T]` transpiles to one returning `*T`, with
      `Option.None` -> nil and `Option.Some(x)` -> the address of x.
- [ ] A statement-position Option match transpiles to an `if o := …; o != nil`
      split that builds and vets.
- [ ] Every `features/03-result` and `features/04-option` transpile case passes
      the behavioral tier (build + vet of the generated Go) through the new (AST)
      backend.

## User Interactions

Indirect — exercised via `goal build --engine=ast` and the corpus behavioral
runner. No new user-facing CLI surface.

## Error Handling

A construct outside this feature's scope (notably closed-E Result, which is a
different sum encoding) MUST NOT be silently mis-lowered by the open-E path; the
backend surfaces a descriptive unsupported-construct error for it (deferred to a
later story) rather than emitting wrong code.

## Out of Scope

- Closed-E Result (`Result[T, E]` where E is not `error`) — its sum encoding is
  US-037.
- Value-position match over Result/Option — US-036.
- The `?` postfix propagation operator — US-035.
- Exact golden (byte-for-byte) parity — judged behaviorally here; goldens are
  regenerated in US-042.

## Open Questions

None — the lowering shapes are fixed by the legacy splice reference
(`internal/pass/result.go`, `internal/pass/option.go`) and the existing
`features/03-result` / `features/04-option` goldens.
