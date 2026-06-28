# Run Doctests Behaviorally — Business Specification

## Overview

Goal's golden corpus has a behavioral tier that builds and vets the Go emitted
for each transpile case (US-007), and a doctest tier that compares the emitted
`_test.go` sidecar against a checked-in golden (US-005). Neither actually runs
the doctest. This feature adds a behavioral tier that EXECUTES doctest sidecars:
for every doctest-bearing corpus case it compiles the generated package together
with its generated test file and runs `go test`, so a doctest that would fail at
runtime turns the suite red.

## Functional Requirements

### FR-1: Execute each doctest-bearing case
For every corpus case classified as a doctest case, the suite SHALL transpile
the input, write the generated package file and the generated test sidecar into
a single isolated temporary module, and run `go test` against that module.

### FR-2: Behavioral assertion
A test SHALL assert that every doctest-bearing case passes `go test` in its temp
module. If any doctest fails at runtime, or the package/test fails to compile,
the test SHALL fail and identify the offending case.

### FR-3: Non-destructive
The runner SHALL NOT modify, move, or create files in the source tree; all work
happens inside an OS temp directory that is cleaned up afterward.

### FR-4: Loud empty-corpus guard
If the manifest yields zero doctest cases, the test SHALL fail loudly rather
than silently passing.

## Acceptance Criteria

- [ ] For each case that has a doctest sidecar, the runner writes both files to a
      temp module and runs `go test` on it.
- [ ] A test asserts every doctest-bearing case passes `go test` in its temp
      module.
- [ ] An empty doctest set fails the test loudly (no silent green).
- [ ] `go build ./...`, `go vet ./...`, and `go test ./... -count=1` are green.

## User Interactions

None directly user-facing. Exercised via `go test ./internal/corpus/`.

## Error Handling

On a missing input, transpile failure, empty sidecar, temp-module write failure,
or `go test` failure, the runner returns a descriptive, case-identified error
including the go tool's combined output.

## Out of Scope

- Re-running cases whose golden is the main Go output (those are covered by the
  compile tier).
- Package-mode multi-file fixtures (US-009).
- Any front-end change; this judges whichever Transpiler is supplied.

## Open Questions

None.
