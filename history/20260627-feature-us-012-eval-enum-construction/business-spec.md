# Business Spec — US-012 Eval enum construction

## Outcome

A goal program running under the goscript interpreter can construct enum values
and the interpreter represents them as tagged unions at runtime.

## Requirements

- The interpreter constructs enum variant values via the `Enum.Variant(...)`
  surface:
  - Data-less variants: `Status.Pending`.
  - Payload-carrying variants: `Status.Active(since: now())`,
    `Status.Cancelled(reason: "timeout", at: now())`.
- Each constructed value is a tagged union recording the enum type, the variant
  tag, and the named payload fields.
- The payload fields of a constructed variant can be read back by name.
- Constructing an unknown enum or an unknown variant of a known enum is a loud,
  descriptive refusal — never a silent nil or wrong tag.

## Acceptance Criteria

- The interpreter constructs enum variant values (data-less and payload-carrying)
  via `Enum.Variant(...)` into tagged-union Values and reads their payload fields.
- A unit test over a 01-enums-shaped program constructs each variant kind and
  asserts the tag and field values.
