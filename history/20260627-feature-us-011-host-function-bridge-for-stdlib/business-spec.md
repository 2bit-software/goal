# Business Spec — US-011 Host-function bridge for stdlib

## Goal

A goal program interpreted by goscript can call a curated set of Go standard
library functions and observe their native behavior. Programs that call an
imported symbol the interpreter does not yet shim fail loudly and by name,
never silently producing a nil/empty result.

## Acceptance Criteria

1. internal/interp provides a host-function registry resolving at least
   `fmt.Sprintf`, `fmt.Sprint`, `fmt.Println`, `fmt.Errorf`, and `errors.New`
   to native Go implementations; an unresolved imported call produces a
   located, named error rather than a silent nil.
2. A unit test runs a goal program calling `fmt.Sprintf` and asserts the
   produced string, and asserts an unregistered imported call yields a
   descriptive error naming the missing symbol.

## Constraints

- Behavior only; the *what* is: the listed symbols work, unknown imported
  symbols are an explicit, named refusal.
- Refusals must name the missing `pkg.Symbol` and include a source location.
