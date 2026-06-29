# US-001 Rename backend package to selfhost — Business Specification

## Overview

Mirror the compiler's largest package, `internal/backend`, into `selfhost/backend`
as goal source so the verbatim-rename step of the self-host idiomatic effort covers
it. This is a faithful copy (Go is a superset of goal); no idiomatic rewrites.

## Functional Requirements

### FR-1: Mirrored package
`selfhost/backend` contains the 6 non-test files of `internal/backend`
(arity, backend, doctest, emit, lower, package) as `.goal` files, byte-equivalent
to the originals except for any reserved-word identifier renames (none are needed).

### FR-2: Compile gate
The transpile-and-build smoke gate (`selfhost.BuildTranspiled`) transpiles
`selfhost/backend` plus its ported dependency closure and the generated Go builds.

### FR-3: Behavioral gate
The self-contained subset of the existing backend tests passes against the
transpiled package via `selfhost.BuildAndTest`.

## Acceptance Criteria

- [ ] `selfhost/backend` holds backend as `.goal` files mirroring `internal/backend`.
- [ ] `selfhost.BuildTranspiled` over the backend dep closure compiles.
- [ ] The fixture-free backend tests pass against the transpiled package.
- [ ] `task check`, `task build`, `task fixpoint` are green.

## User Interactions

None directly user-facing. Exercised through `go test ./internal/selfhost` and the
fixpoint/bootstrap targets.

## Error Handling

Gate failures surface as descriptive, package-identified errors from the selfhost
harness (transpile failure, invalid generated Go, build/test failure).

## Out of Scope

- Idiomatic upgrades (Result/?, enum, match, sealed interface) — US-004/US-011.
- Porting typecheck (US-002) and wiring main onto selfhost packages (US-003).
- Tests requiring repo-relative fixtures or the corpus harness.
