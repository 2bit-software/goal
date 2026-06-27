# Research — US-002 interp value model

## Findings (internal, codebase-grounded)

- No `internal/interp` package exists yet; this story creates it. (Confidence: High)
- House style for a new dependency-free runtime package is established by
  `internal/cap`: package doc comment referencing REWRITE-ARCHITECTURE.md, a kind
  iota + String() switch, stdlib-only, tests in stdlib `testing` (NO testify).
- Tagged-union discriminants in the source language are string tags: enum cases
  and the built-in sums use names like `Ok`/`Err`/`Some`/`None` (see
  internal/sema tests). So `Variant.Tag` should be a string discriminant; the
  same `Variant{TypeID, Tag, Fields}` shape serves enum, Result, and Option
  uniformly. (Confidence: High)
- The Go backend lowers Result->(T,error) and Option->*T (internal/backend
  lower.go). The interpreter must NOT reuse that — the universal tagged union is
  the explicit, separate encoding (REWRITE-ARCHITECTURE.md §4). (Confidence: High)

## Design decision

A single concrete `Value` struct with a `Kind` discriminant (Int, Float, String,
Bool, Nil, Struct, Slice, Map, Func, Variant) holding the corresponding Go-native
payload. `Variant{TypeID string, Tag string, Fields map[string]Value}` is the
universal tagged union, embedded in a Value of Kind=Variant, with a Field(name)
accessor. Provide `Equal(Value) bool` and `String() string`.

## Confidence: High
## Open questions: none blocking — function payload representation can be a
minimal opaque carrier now (callable wiring is US-004+).
