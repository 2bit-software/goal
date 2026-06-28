# Business Spec — US-040 Emit doctest sidecars on new path

As a goal developer, I need doctests extracted and emitted by the new backend so
executable docs survive the front-end change.

## Acceptance Criteria
- The new backend extracts `///` doctests and emits the `_test.go` sidecar
  lowered through the same path as function bodies.
- The 11-doctests cases pass the doctest tier through the new backend.

## Constraints
- Zero-dependency: stdlib `testing` only (no testify).
- A doctest over goal-specific values (enum variants, keyed literals,
  Result/Option constructors) must lower exactly as a function body does.
