# Derive / From Lowering — Business Specification

## Overview

goal lets a developer declare type-directed struct conversions: leaf conversions
with `from func`, and whole-struct conversions with `derive func`. The new AST
backend must lower these into ordinary Go so the conversion runs. A `derive func`
is expanded field-by-field from a source struct to a target struct, resolving each
target field from a same-named source field through identity, a registered leaf
conversion, or built-in container recursion. This story brings that lowering to the
AST engine; the `from func` modifier is already stripped by the backend.

## Functional Requirements

### FR-1: From-func leaves emit as plain functions
A `from func name(...) ...` SHALL emit as an ordinary Go function with the `from`
modifier removed and its body unchanged.

### FR-2: Bodyless total derive
A `derive func name(src S) T` with no fallible leaf SHALL emit a function that
declares `var out T`, assigns every target field from its same-named source field,
and returns `out`. A field of identical type is a direct assignment; a field
requiring conversion uses the registered leaf conversion.

### FR-3: Fallible derive threads the error
When any target field is filled by a fallible leaf conversion (one returning
`(T, error)`), the derive function SHALL return `(T, error)`, evaluate the fallible
conversion into a temporary, early-return the error on failure, and otherwise assign
the converted value.

### FR-4: Container recursion
A target field whose conversion is element-wise over a slice, array, map, or
pointer/Option SHALL be filled by recursing on the element conversion (e.g.
`[]A -> []B` builds a new slice and converts each element via the registered
`A -> B`).

### FR-5: Bodied derive with overrides
A `derive func` with a body returning a composite literal SHALL honor: an explicit
`Field: expr` override (emitted verbatim), a `Field: _` skip (target field left at
its zero), and a `...derive(src)` element that fills every remaining target field by
the rules above.

### FR-6: Resolved-type field matching
Field-to-field resolution SHALL use the resolved struct fields and conversion
registry (semantic facts), not ad-hoc parsing of source text.

## Acceptance Criteria

- [ ] A `from func` leaf appears in the generated Go as a plain `func` with no
      `from` keyword.
- [ ] The 12-derive-convert example `slice.goal` (slice container recursion)
      transpiles and the generated Go builds and vets cleanly.
- [ ] The 12-derive-convert example `from_storage.goal` (fallible leaf threaded
      through the derive) transpiles and the generated Go builds and vets cleanly.
- [ ] The 12-derive-convert example `to_storage.goal` (overrides, `_` skip, and
      `...derive(src)`) transpiles and the generated Go builds and vets cleanly.
- [ ] A target field with no same-named, resolvable source field produces a
      descriptive error rather than a silently-zeroed field.

## User Interactions

Developers write `from func` / `derive func` declarations in `.goal` source and run
the transpiler on the AST engine. No new CLI surface is introduced.

## Error Handling

An unresolvable target field (no same-named source field, or no registered
conversion for its type pair) SHALL fail transpilation with a located message
naming the field and the missing conversion — never emit a silent zero.

## Out of Scope

- Foreign/cross-package derive fixtures (exercised through the package runner on the
  splice engine), which are not part of the AST-backend file cases here.
- Exact byte-for-byte golden parity with the splice engine (a later regeneration
  story); generated temporary names differ.

## Open Questions

- None. The behavior is fully specified by the existing feature 12 examples and the
  known-good splice lowering.
