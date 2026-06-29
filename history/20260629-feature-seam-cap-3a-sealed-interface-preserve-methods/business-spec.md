# Sealed interfaces preserve method signatures — Business Specification

## Overview

A goal `sealed interface` is a closed-implementor interface that lowers to a Go
interface carrying a synthesized unexported marker method. Today the transpiler
keeps only that marker and silently drops any method signatures the author
declared in the interface body. This makes it impossible to seal an interface
whose very purpose is its methods (e.g. an AST `Node` with `Pos()`/`End()`).

This feature makes a sealed interface preserve its declared method signatures in
the emitted Go interface, alongside the marker method, so sealed interfaces can
carry real method sets.

## Functional Requirements

### FR-1: Declared methods are preserved
When a sealed interface declares method signatures, the emitted Go interface SHALL
contain each declared method signature (name, parameters, results).

### FR-2: Marker method is retained
The emitted Go interface SHALL still contain the synthesized unexported marker
method (`isName()`), keeping satisfaction closed to same-package implementors.

### FR-3: Empty sealed interfaces are unchanged
A sealed interface declaring no methods SHALL continue to emit the compact
`type Name interface{ isName() }` form, byte-identical to today.

### FR-4: Implementors compile and methods are callable
A type that declares `implements Name` and provides the declared methods SHALL
compile, and those methods SHALL be callable through a value of the interface type.

### FR-5: Live and self-hosted transpilers agree
The behavior SHALL be identical in the live Go transpiler and the self-hosted
.goal mirror, so self-host fixpoint self-consistency is preserved.

## Acceptance Criteria

- [ ] A sealed interface declaring `Pos() Position` and `End() Position` emits a Go
      interface containing both method signatures and the `isName()` marker.
- [ ] An empty-body sealed interface still emits `type Name interface{ isName() }`.
- [ ] An implementor providing the declared methods builds, and a function that
      calls a declared method through the interface value builds and runs.
- [ ] `task check`, `task build`, `task fixpoint` are all green.
- [ ] The corpus behavioral tier is unchanged.

## User Interactions

No new surface. Authors write a `sealed interface Name { Method(...) ... }` and the
emitted Go interface now includes those methods.

## Error Handling

No new error states. A sealed interface with no name still fails as before.

## Out of Scope

- Type-pattern `match` over a sealed interface scrutinee (CAP-3b).
- Building an implementor registry / exhaustiveness checking (CAP-3b).
- Cross-.goal-package propagation of implementor sets (CAP-3c).
- Sealing ast.Node itself (later SEAM-004).

## Open Questions

None — root cause and fix are confirmed and localized.
