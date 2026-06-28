# US-018 Eval implements dispatch — Business Specification

## Overview

The goscript tree-walking interpreter (internal/interp) must honor `implements`:
a goal value that satisfies an interface, when used where that interface is
expected, dispatches an interface method call to the value's own concrete
method. Because the interpreter erases static types at runtime, an
interface-typed binding simply holds the concrete value; dispatch is structural
on the concrete type. This story proves that sealed/ordinary interface method
dispatch runs correctly at runtime.

## Functional Requirements

### FR-1: Concrete dispatch through an interface
When an interface-typed binding holds a concrete value, calling an interface
method invokes that value's own concrete method implementation.

### FR-2: Differently-typed dispatch
Calling the same interface method on differently-typed values that each satisfy
the interface runs each type's own concrete implementation, selected by the
runtime value's type.

### FR-3: Value- and pointer-receiver concrete methods
Both value-receiver and pointer-receiver concrete methods dispatch through the
interface. A pointer-receiver method observes and may mutate the underlying
value; a value-receiver method operates on a copy.

### FR-4: Dispatch from interface collections
A method called on an interface value obtained from a collection (e.g. an
element of a heterogeneous slice of interface values) dispatches to that
element's concrete implementation.

## Acceptance Criteria

- [ ] The interpreter honors `implements`: a struct satisfying a sealed
      interface is dispatched through the interface, calling the correct
      concrete method at runtime.
- [ ] A unit test over a 07-implements shape calls an interface method on
      differently-typed values and asserts each concrete implementation runs.
- [ ] The same test exercises both a value-receiver and a pointer-receiver
      concrete method dispatched through the interface.
- [ ] Project verify gates stay green: `go build ./...`, `go vet ./...`,
      `go test ./... -count=1`.

## User Interactions

Authors write goal programs with `implements` clauses and interface-typed
parameters/locals/collections; running the program under the interpreter
dispatches interface method calls to the concrete implementations. There is no
new CLI surface in this story.

## Error Handling

Calling a method that the concrete value's type does not implement surfaces a
descriptive interpreter error rather than a silent nil (such programs are
already rejected statically by sema; the runtime stays loud, not silent).

## Out of Scope

- The address-of operator `&` and general first-class pointers (a broader
  feature not required here). Pointer-receiver dispatch is covered without `&`
  because goal struct values are addressable through the shared underlying
  struct value.
- Interface method dispatch on enum/sum Variant values — the 07-implements shape
  uses ordinary struct implementers.

## Open Questions

None. The dispatch seam (US-010 method registry) already exists; this story is
the conformance proof that interface dispatch rides on it.
