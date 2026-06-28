# Business Spec — US-010 Eval builtins and methods

As a runtime author, I need builtins and method dispatch so idiomatic
Go-subset code runs under interpretation.

## Acceptance Criteria

1. The interpreter implements the `len`, `append`, `make`, and `panic`
   builtins and dispatches both value-receiver and pointer-receiver methods
   declared on goal types.
2. A unit test asserts `append`/`len` on a slice, `make` of a map, a recovered
   panic, and a method call mutating a pointer receiver.

## Constraints

- Zero external dependencies; tests use stdlib `testing` only (no testify).
- The interpreter must keep its dependency surface clean (no go/types,
  internal/backend, or internal/typecheck) for the later US-022 gate.
- Unsupported/edge forms must be loud, descriptive refusals, never silent nils.
