# Eval derive and from — Business Specification

## Overview

The goscript tree-walking interpreter must execute generated `derive func`
conversions. A `derive func` declares a structural conversion from a source
struct to a target struct; the runtime must construct the target value by
sourcing each target field from its same-named source field, applying registered
`from func` conversions where the source and target field types differ. This lets
goal programs that rely on derived conversions run identically under
interpretation as under Go transpilation.

## Functional Requirements

### FR-1: Derived conversion produces the target struct
Calling a `derive func` at runtime SHALL produce a target struct value whose
every declared field is populated. A target field whose source field has the same
type SHALL be copied through unchanged (identity).

### FR-2: Bridged fields use the from-registry
When a target field's type differs from its same-named source field's type, the
runtime SHALL apply the registered `from func` conversion for that
(source-type, target-type) pair. Nested in-file struct fields, and slice/array/map
fields whose element types bridge, SHALL be converted recursively using the same
rules.

### FR-3: Fallible conversions thread the error
A `derive func` declared to return `(T, error)` SHALL apply fallible registered
conversions; if any such conversion returns a non-nil error, the derivation SHALL
short-circuit and return that error as its second result rather than a partially
built value used as success.

### FR-4: Bodied overrides are honored
A `derive func` with a body MAY override a target field with an explicit
expression, skip a field with `_` (leaving it at its safe zero), and fill the
remaining fields implicitly via `...derive(src)`. The runtime SHALL evaluate
override expressions against the source binding and apply them before the implicit
fill.

### FR-5: Unsourced or unconvertible fields fail loudly
A target field that is neither overridden nor sourced from a same-named source
field, or whose conversion cannot be resolved, SHALL produce a descriptive,
located refusal — never a silently zeroed field.

## Acceptance Criteria

- [ ] The interpreter evaluates `derive func` conversions field-by-field using the
      resolved sema struct fields and from-registry, applying from-registry
      conversions for bridged fields.
- [ ] A unit test over a 12-derive-convert shape asserts a derived conversion
      produces the expected target struct (identity field, a registry-bridged
      field, and a nested struct field all correct).
- [ ] A fallible derive returns the converted target on success and propagates a
      non-nil conversion error on failure.
- [ ] An unsourced/unconvertible target field yields a descriptive error, not a
      silent zero.
- [ ] All project verify gates stay green: `go build ./...`, `go vet ./...`,
      `go test ./... -count=1`.

## User Interactions

No direct user surface. The behavior is observed by executing a goal program
containing a `derive func` through the interpreter and inspecting the produced
value (the unit test harness drives `New(file, info)` + a call to the derive).

## Error Handling

- Unknown target struct, unsourced target field, or unresolvable field conversion:
  a descriptive interpreter error naming the derive and the field.
- A fallible conversion error: returned as the derive's second (error) result.

## Out of Scope

- Pointer (`*T`) and `Option[T]` field recursion in a derivation — the interpreter
  models no address-of operator; such a field is refused loudly, deferred honestly.
- Cross-package / foreign derive conversions (still covered by the package-mode
  corpus runner, not this story).
- Any change to the Go transpiler backend's derive lowering.

## Open Questions

None — the runtime strategy mirrors the established Go backend lowering over the
same sema facts.
