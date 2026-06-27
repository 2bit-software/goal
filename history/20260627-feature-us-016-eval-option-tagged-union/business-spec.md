# Eval Option as tagged union — Business Specification

## Overview

The goscript interpreter must represent `Option` values using the same universal
tagged-union encoding it uses for enums and `Result`, rather than the Go
backend's `*T` optimization. This keeps the runtime model uniform: every sum type
(enum, Result, Option) is one `Variant{TypeID, Tag, Fields}` shape, and `match`
dispatches over all of them by tag.

## Functional Requirements

### FR-1: Construct Option.Some
The interpreter SHALL evaluate `Option.Some(x)` to a present tagged-union value
(TypeID "Option", Tag "Some") carrying `x` as its single payload.

### FR-2: Construct Option.None
The interpreter SHALL evaluate `Option.None` to an absent tagged-union value
(TypeID "Option", Tag "None") carrying no payload.

### FR-3: Match over Option
A statement- or value-position `match` over an `Option` SHALL run the `Some` arm
for a present value — binding the UNWRAPPED inner value to the arm binding — and
the `None` arm for an absent value.

## Acceptance Criteria

- [ ] `Option.Some(x)` evaluates to a variant tagged "Some" under TypeID "Option"
      whose single payload is `x`.
- [ ] `Option.None` evaluates to a variant tagged "None" under TypeID "Option"
      with no payload.
- [ ] A `match` over a `Some` value runs the `Some` arm and binds the unwrapped
      inner value (read directly, not via `.field`).
- [ ] A `match` over a `None` value runs the `None` arm.
- [ ] A unit test over a 04-option shape constructs `Some` and `None` and asserts
      the match arms behave correctly.

## User Interactions

A goal author writes `Option.Some(v)` / `Option.None` and consumes them with
`match`. No CLI or API surface changes.

## Error Handling

An unknown `Option` constructor name or a `Some` call with other than exactly one
argument SHALL be a located, descriptive refusal — never a silent value, matching
the `Result` constructor's stance.

## Out of Scope

- Postfix `?` unwinding over Option (US-017).
- Any `*T` runtime optimization — explicitly excluded.
- Non-`match` Option consumers.

## Open Questions

None — the encoding mirrors the shipped Result implementation (US-015).
