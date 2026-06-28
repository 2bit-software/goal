# Technical Requirements & Research — US-002

## From the PRD notes

REWRITE-ARCHITECTURE.md §4: the interpreter IGNORES the Go-codegen optimizations
(Result->(T,error), Option->*T) and uses the universal tagged-union. Do not reuse
backend/go's lowering.

## Implementation hints

- New package `internal/interp` (sibling to internal/cap, internal/sema).
- Mirror the house style of internal/cap: small, dependency-free, doc-commented
  package; stdlib only; tests in stdlib `testing` (NO testify).
- Value model: a single concrete `Value` type with a kind discriminant covering
  int, float, string, bool, nil, struct, slice, map, function, and variant.
- `Variant{TypeID, Tag, Fields}` is the universal tagged union: TypeID names the
  declared type identity, Tag is the variant discriminant (e.g. "Ok"/"Err"/"Some"/
  "None"/an enum case), Fields are named payload values readable by name.
- Provide Value equality and a String() rendering for tests and later eval output.

## No specific technical requirements beyond the above were mandated by the user.
