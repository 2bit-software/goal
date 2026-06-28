# Eval value-position match — Business Specification

## Overview

The goscript tree-walking interpreter consumes goal's sum types via `match`.
Statement-position `match` (dispatch-and-run) already works. This feature adds
value-position `match`: a `match` used as an expression that yields the selected
arm's value. Value-position match is one of only two genuinely non-Go runtime
mechanics, so the interpreter must support it directly.

## Functional Requirements

### FR-1: match yields a value in expression position

When a `match` appears in a value position, the interpreter SHALL evaluate the
scrutinee, dispatch on its variant tag to the matching arm, bind the matched
payload, evaluate that arm's body as an expression, and produce that value as the
result of the `match`.

### FR-2: the three value positions are supported

The interpreter SHALL evaluate value-position match in all three forms:
`return match { ... }`, `x := match { ... }`, and `var x = match { ... }`.

### FR-3: dispatch is uniform with statement-position match

Arm selection SHALL key on the variant tag, fall back to a `_` rest arm when no
variant arm matches, and bind any arm payload binding the same way as
statement-position match.

### FR-4: defensive default stays loud

If a scrutinee's tag matches no arm (a state a sema-proven-exhaustive program
cannot reach), the interpreter SHALL raise a loud `unreachable` panic, never
silently produce a zero or empty value.

## Acceptance Criteria

- [ ] A function whose body is `return match { ... }` returns the matched arm's
      value for each input variant.
- [ ] A variable bound with `x := match { ... }` holds the matched arm's value
      for each input variant.
- [ ] A variable declared with `var x = match { ... }` holds the matched arm's
      value for each input variant.
- [ ] A payload-carrying arm can compute its value from the bound payload (e.g.
      `Circle{radius} => radius * radius`).
- [ ] A `_` rest arm supplies the value when no variant arm matches.
- [ ] A match whose scrutinee tag matches no arm raises an `unreachable` panic.
- [ ] Statement-position match behaviour is unchanged.

## User Interactions

No direct UI/CLI surface in this story; the behaviour is exercised through goal
programs run by the interpreter and asserted in unit tests.

## Error Handling

- A non-variant scrutinee is a descriptive, named refusal.
- A non-expression arm body in value position is a descriptive refusal.
- A tag matching no arm is a loud `unreachable` panic (not a silent value).

## Out of Scope

- Result/Option represented as tagged unions (US-015 / US-016).
- Postfix `?` early-return unwinding (US-017).
- Non-`match` expression features.

## Open Questions

None — the design mirrors the shipped statement-position match (US-013).
