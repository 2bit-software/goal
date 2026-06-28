# Business Spec — US-013 Eval statement-position match

## Goal

As a runtime author, I need match dispatch so sum types are consumed by tag at
runtime.

## Requirements

- The interpreter evaluates statement-position `match`: it dispatches on the
  variant tag, binds the matched payload into scope, runs the selected arm, and
  panics `unreachable` on the defensive default of a proven-exhaustive match.

## Acceptance Criteria

- A unit test over a 02-match-shaped program asserts the correct arm runs for
  each variant and that the default arm is unreachable.

## Constraints

- Preserve the "erase the guarantee, panic-not-silent" stance: the
  proven-exhaustive default arm is a loud panic, never a silent fall-through.
- Match consumes the universal tagged-union Variant value produced by enum
  construction (US-012), not any Go-lowered form.
