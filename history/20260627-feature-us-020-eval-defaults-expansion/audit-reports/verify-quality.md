# Verify: Quality — US-020

## Findings

No CRITICAL. No MAJOR.

### MINOR-1: defined-type-over-primitive zero
`zeroValue` resolves primitives, ref types, slices, and named in-file structs,
but a defined type over a primitive (`type Role int`) is neither a primitive
keyword nor a struct, so its default would be NilVal rather than IntVal(0). The
front-end's static field check (sema CheckFields) determines safety; the
feature-08 fixtures set such fields explicitly, so this path is unexercised and
not required by the spec. Could be addressed later by resolving defined-type
underlyings from the AST File (as the backend's buildTypeDecls does), if a
fixture ever defaults one.

### MINOR-2: array (`[N]T`) zero
Falls through to the named-type branch -> NilVal. No fixture uses an array
field. Acceptable; not required.

## Error handling
- Non-defaults spread -> located, descriptive refusal naming ...defaults.
- `...defaults` on an unresolved struct type -> located "unknown struct type"
  refusal, never a silent empty struct.
Both match the spec's error-handling section and are tested.

## Behavior fidelity
- The fill runs after all explicit elements are collected, so `...defaults` is
  position-independent and never overwrites an explicit field — verified by a
  spread-before-explicit test. Matches FR-2.
- Zero mapping mirrors the Go backend's zeroLit, keeping interpreter and
  transpiler behavior aligned.

## Assumptions
- Same as the acceptance report: empty-slice slice zero; defined-type-over-
  primitive defaulting is out of scope.
