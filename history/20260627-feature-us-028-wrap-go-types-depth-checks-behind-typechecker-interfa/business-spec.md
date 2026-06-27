# Wrap go/types depth checks behind interface — Business Specification

## Overview

The goal compiler runs a set of "depth" correctness checks (implements,
must-use, no-zero-value) that today reach for the Go compiler: it transpiles a
goal package to Go and asks `go/types` real type questions. Per the rewrite
architecture (decision 4 / §3.2) this `go/types`-over-lowered-Go path is a
deliberate crutch that must later be replaceable by a native goal type checker
without disturbing the code that requests depth checks. This story introduces a
`TypeChecker` seam so that swap becomes a drop-in.

## Functional Requirements

### FR-1: Depth-checking seam
A `TypeChecker` abstraction SHALL exist that, given a goal package, produces the
depth-check diagnostics. The current transpile-then-`go/types` implementation
SHALL satisfy this abstraction.

### FR-2: Caller indirection
The compiler's depth-check entry point SHALL obtain its diagnostics through the
`TypeChecker` abstraction rather than by reaching for the concrete
implementation directly, so a future implementation can be substituted with no
change at the call site.

### FR-3: Behavior preservation
The diagnostics produced through the abstraction SHALL be identical to those
produced by the existing depth checks (same implements, must-use, and
no-zero-value findings), for both clean and error-bearing packages.

## Acceptance Criteria

- [ ] A `TypeChecker` interface is defined and the existing typecheck (transpile
  then go/types) implements it.
- [ ] A test exercises the depth checks through the interface and the existing
  typecheck cases still pass.
- [ ] The compiler's depth-check entry point resolves its diagnostics through the
  interface value.
- [ ] `go build ./...`, `go vet ./...`, and `go test ./... -count=1` are green.

## User Interactions

No user-visible surface change. `goal check` continues to emit the same depth
diagnostics; the change is internal structure only.

## Error Handling

A transpile/parse failure (a goal-compiler bug) SHALL continue to surface as an
error from the abstraction; genuine user-program type errors SHALL continue to
be tolerated and folded into diagnostics exactly as today.

## Out of Scope

- Any native (non-`go/types`) type checker — deferred until the runtime forces
  it (decision 4).
- Any change to the depth-check logic, messages, codes, or positions.

## Open Questions

None — the seam shape follows the established interface pattern in the codebase
(Backend / Formatter / Transpiler / Checker) and REWRITE-ARCHITECTURE.md §3.2.
