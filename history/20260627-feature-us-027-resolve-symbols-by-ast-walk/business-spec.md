# US-027 Resolve symbols by AST walk — Business Specification

## Overview

The goal compiler currently derives its name-keyed symbol facts (enums, structs,
function signatures, type conversions, methods) by re-scanning a flat token
stream in `internal/analyze`. Because that scanning models structure with
whitespace and brace counting rather than a parsed grammar, it mis-resolves any
construct token scanning cannot faithfully model — most notably a struct field
whose type contains a top-level comma.

This feature introduces a semantic-resolution layer that derives the same
name-keyed facts structurally, by walking the already-parsed goal AST. Because
the facts come off the parse tree, they are correct by construction wherever the
token scanner is not.

## Functional Requirements

### FR-1: Resolve enums
The system SHALL resolve every enum declaration to its name, its variants in
source order, and for each variant its payload fields (name and type), including
membership sets for variant names and per-variant field names.

### FR-2: Resolve structs
The system SHALL resolve every struct type declaration to its ordered fields
(name and type), expanding a shared-type field group (`a, b int`) into one entry
per name.

### FR-3: Resolve function signatures
The system SHALL resolve each function's return signature: its error-propagation
mode (none, open-E Result, closed-E Result, or Option), its success type and
error type where applicable, its return arity, and whether its last result is an
error a `?` can propagate.

### FR-4: Resolve the from-registry
The system SHALL resolve each `from func` and `derive func` conversion, keyed by
its (source type, target type) pair, recording the conversion function name and
whether it is fallible.

### FR-5: Resolve methods
The system SHALL resolve every method declaration under its receiver type name
(pointer and value receivers alike contributing to the same receiver key).

### FR-6: Correct resolution of comma-bearing field types
The system SHALL resolve a struct field whose type contains an embedded
(top-level) comma — for example a func-typed field `cb func(int, string)` — as a
single field with its complete, correct type. This is the case the token-scanning
resolver mishandles.

## Acceptance Criteria

- [ ] Enums, structs, function signatures, the from-registry, and methods are all
  resolved from a parsed goal file.
- [ ] For a representative goal source covering each of those construct kinds, the
  resolved facts match those produced by the existing token-scanning resolver
  (same enum variants/fields, struct fields, signature mode/types/arity,
  from-registry entries, and per-receiver methods).
- [ ] A struct whose field type contains an embedded comma resolves to the correct
  number of fields, each with its complete type — demonstrably correct where the
  token scanner is not.
- [ ] The project's verify gates (`go build ./...`, `go vet ./...`,
  `go test ./... -count=1`) stay green.

## User Interactions

No end-user surface. The capability is consumed internally by the AST back-end
(later stories). It is exercised through unit tests.

## Error Handling

Resolution is total over a well-formed parsed file: an unrecognized or
not-yet-modeled declaration is skipped rather than failing, mirroring the
tolerant behavior of the existing resolver. Result maps are always initialized
(safe to read even when empty).

## Out of Scope

- The correctness checks layered on these facts (match exhaustiveness,
  field-completeness, must-use / implements / `?`-arity) — US-029..US-031.
- Wiring the resolved facts into the back-end emit path — US-032+; the back-end
  continues to use an empty semantic-info value for now.
- Cross-file/package union of facts — single-file resolution mirrors
  `analyze.Build`; package union mirrors a later need.
- Foreign (imported-package) method enrichment.

## Open Questions

None. Scope and parity target (single-file `analyze.Build`) are fixed by the PRD
story and the existing analyze surface.
