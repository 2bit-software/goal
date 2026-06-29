# Generic function declarations — Business Specification

## Overview

goal must accept top-level generic function declarations and transpile them
to valid Go. Today the parser rejects `func Name[...]( ... )` with
`expected (, found [`, so any compiler code using a generic function cannot
be expressed. This feature closes that gap.

## Functional Requirements

### FR-1: Parse generic functions
A top-level function with a type-parameter list after its name parses with
no error: `func Identity[T any](x T) T { return x }`.

### FR-2: Constrained type parameters
A constrained type parameter is accepted, e.g. `func Keys[K comparable, V any](m map[K]V) []K`.

### FR-3: Faithful transpile
The transpiled Go preserves the type-parameter list and is accepted by
`go build`.

### FR-4: Non-generic functions unchanged
Functions and methods without a type-parameter list behave exactly as before.

## Acceptance Criteria

- [ ] `func Identity[T any](x T) T { return x }` parses with no diagnostic.
- [ ] A constrained param (`[T comparable]`) parses and transpiles.
- [ ] The transpiled generic function compiles with `go build`.
- [ ] Existing non-generic function transpilation is unchanged (full suite green).

## User Interactions

Authoring goal source; no new CLI surface.

## Error Handling

A malformed type-parameter list reports a parse diagnostic as other
malformed constructs do; well-formed lists produce no error.

## Out of Scope

- Generic methods (type params on a method with a receiver) — Go disallows them.
- Type inference changes; sema/typecheck behavior beyond passing the list through.

## Open Questions

None.
