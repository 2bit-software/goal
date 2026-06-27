# US-009 Eval composite types — Business Specification

## Overview

The goscript tree-walking interpreter can currently evaluate primitives,
operators, variables, functions, and control flow, but not aggregate data.
This feature adds evaluation of the three core composite types — structs,
slices, and maps — so real data structures can be built, read, mutated, and
iterated under interpretation, following Go semantics for the supported subset.

## Functional Requirements

### FR-1: Struct composite literals and field access
A keyed struct composite literal (`T{field: value, ...}`) evaluates to a struct
value carrying the type name and the named field values. A field selector
(`x.field`) on a struct value reads that field's current value.

### FR-2: Slice literals and indexing
A slice composite literal (`[]E{a, b, c}`) evaluates to an ordered slice value.
An index expression (`s[i]`) reads the element at integer index `i`; an
out-of-range index is a descriptive error.

### FR-3: Map literals, indexing, and key assignment
A map composite literal (`map[string]V{k: v, ...}`) evaluates to a map value.
An index expression (`m[k]`) reads the value at key `k` (the zero/absent read is
a defined result). A key assignment (`m[k] = v`) inserts or updates the entry.

### FR-4: Index and field assignment targets
Assignment targets may be an index expression (`s[i] = v`, `m[k] = v`) or a
struct field selector (`x.field = v`), mutating the underlying collection or
struct in place with Go reference semantics.

### FR-5: Range-for over slices and maps
A `for k, v := range x` statement iterates a slice (key = integer index, value =
element, in order) or a map (key, value per entry). The blank identifier and the
key-only form are honored; `:=` binds fresh loop variables.

## Acceptance Criteria

- [ ] A struct composite literal builds a struct whose fields read back the
      assigned values via field access.
- [ ] A slice literal builds a slice whose elements read back by index.
- [ ] A map literal builds a map whose values read back by key, and a key
      assignment updates the map.
- [ ] Ranging over a slice yields ascending indices with their elements.
- [ ] Ranging over a map visits each key with its value.
- [ ] A unit test builds and reads structs, slices, and maps, ranges over a
      slice and a map, and asserts the collected results.
- [ ] An out-of-range slice index and an unsupported target/key produce a
      descriptive error, never a silent nil or a panic.

## User Interactions

None directly user-facing; this is interpreter-internal behavior exercised by
goal programs and unit tests.

## Error Handling

Out-of-range indexing, indexing a non-collection, field access on a non-struct,
a non-string map key (v1 maps are string-keyed), and ranging a non-iterable are
each a located, descriptive error consistent with the rest of the interpreter.

## Out of Scope

- Non-string map keys (deferred per the v1 value model).
- Positional (unkeyed) struct composite literals.
- Slice expressions (`s[lo:hi]`), `append`/`len`/`make` builtins (US-010).
- Pointers to composites and method dispatch (US-010).
- Enum/Result/Option variant values (US-012+).

## Open Questions

None.
