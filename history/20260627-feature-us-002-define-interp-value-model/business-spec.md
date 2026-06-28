# Interp Value Model — Business Specification

## Overview

The goscript tree-walking interpreter needs ONE uniform runtime value
representation so every runtime construct — primitives, composites, and the
language's sum types (enum, Result, Option) — shares a single encoding. This
specification defines that value model. The encoding is deliberately distinct
from the Go backend's optimizations (which lower Result to `(T, error)` and
Option to `*T`): the interpreter uses a single universal tagged union for all
sum types.

## Functional Requirements

### FR-1: Primitive values
The model SHALL represent each primitive runtime value: int, float, string,
bool, and nil.

### FR-2: Composite values
The model SHALL represent struct, slice, and map runtime values, and a function
value.

### FR-3: Universal tagged union
The model SHALL provide a single tagged-union representation carrying a type
identity, a tag (discriminant), and named payload fields. This one representation
SHALL be used uniformly for enum variants, Result (Ok/Err), and Option
(Some/None) — none of these is special-cased and none reuses the Go backend's
optimized encodings.

### FR-4: Field read-back
A tagged-union value's payload fields SHALL be readable by name.

### FR-5: Equality
Two values SHALL be comparable for equality, including tagged-union values
(equal type identity, tag, and fields).

### FR-6: Rendering
Every value SHALL produce a readable string rendering.

## Acceptance Criteria

- [ ] A Value type exists covering int, float, string, bool, nil, struct, slice,
      map, and function.
- [ ] A universal tagged-union value carrying type identity, tag, and named
      fields exists and is used uniformly for enum, Result, and Option.
- [ ] A tagged-union payload field can be read back by name.
- [ ] Constructing a tagged-union value and each primitive/composite value
      succeeds.
- [ ] Value equality returns true for equal values and false for differing ones,
      including tagged-union values.
- [ ] Each value renders to a non-empty, readable string.

## User Interactions

None directly. This is a runtime-internal data model consumed by later
interpreter stories (environment, evaluation, sum-type construction/dispatch).

## Error Handling

- Reading a non-existent field from a tagged-union value SHALL report "not
  present" to the caller rather than panicking.

## Out of Scope

- Evaluation, environments, and any execution behavior (later stories).
- Callable wiring of function values (the function value is a minimal carrier
  here; binding/calling is US-004+).
- Reusing or interoperating with the Go backend's lowered encodings.

## Open Questions

- None blocking.
