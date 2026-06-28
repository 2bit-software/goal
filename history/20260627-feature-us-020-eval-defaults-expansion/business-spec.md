# Eval defaults expansion — Business Specification

## Overview

The goal language's no-zero-value feature lets a struct composite literal opt
its unset fields into their zero values with a `...defaults` spread, instead of
listing every field. The goscript tree-walking interpreter must honor this at
runtime: when it constructs a struct value from a literal containing
`...defaults`, it fills every field the author did not set explicitly with that
field's safe zero value, while preserving every field the author did set.

This makes no-zero-value construction behave under interpretation exactly as it
does after transpilation to Go.

## Functional Requirements

### FR-1: Defaults fill omitted fields

When a struct composite literal contains a `...defaults` element, the
interpreter SHALL fill each declared field of the struct that the literal did
not set explicitly with that field's zero value.

### FR-2: Explicit fields are preserved

The interpreter SHALL leave every explicitly set field at its given value; a
`...defaults` element SHALL NOT overwrite an explicitly provided field,
regardless of whether `...defaults` appears before or after that field.

### FR-3: Safe zero values

The zero value the interpreter fills SHALL match the field's declared type:
the empty string for string, false for bool, 0 for integer and float numeric
types, the nil value for reference types (pointer, map, channel, function,
method-bearing interface), an empty slice for a slice type, and a recursively
zero-filled struct for a named struct type.

### FR-4: Non-defaults spread is refused

A spread element other than `...defaults` (e.g. `...derive`) is NOT part of this
feature; the interpreter SHALL refuse it with a descriptive, located error
rather than silently ignoring it or producing a wrong value.

## Acceptance Criteria

- [ ] A struct literal with `...defaults` that omits primitive fields yields a
      struct whose omitted string/bool/int fields are "", false, and 0.
- [ ] Fields set explicitly in the same literal keep their explicit values.
- [ ] Omitting a named-struct field via `...defaults` yields a zero-valued
      instance of that struct (its own fields zeroed).
- [ ] Omitting a slice field via `...defaults` yields an empty slice.
- [ ] A `...derive` (non-defaults) spread in a struct literal raises a
      descriptive error, not a silent or wrong value.
- [ ] A unit test over an 08-no-zero-value/defaults shape demonstrates the
      above.

## User Interactions

No new user-facing surface. Authors write `...defaults` in goal source exactly
as the no-zero-value feature already defines; this story makes that source run
correctly under the interpreter engine.

## Error Handling

- An unsupported spread (non-`defaults`) produces a descriptive, located error.
- A `...defaults` on a struct type whose fields are unknown to the front-end
  produces a descriptive error rather than a silent empty struct. (In a valid
  program this cannot occur — the front-end resolves the struct's fields.)

## Out of Scope

- `...derive` spread expansion (a separate story, US-021).
- The static guarantee that unsafe-zero fields must be set explicitly — that is
  enforced by the front-end (internal/sema), not the interpreter.
- Any change to the Go transpilation backend.

## Open Questions

- None. The semantics are pinned by the existing front-end (sema field check)
  and Go backend (`zeroLit`) behavior; this story mirrors them at runtime.
