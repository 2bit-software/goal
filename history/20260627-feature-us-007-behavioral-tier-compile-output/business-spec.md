# Add behavioral tier: compile output — Business Specification

## Overview

The corpus suite currently judges transpile conformance by exact (gofmt-
normalized) text comparison against a golden. This story adds a *behavioral*
tier: for every transpile case, the generated Go is proven to actually compile
and pass `go vet`. Conformance is then judged by behavior, not by Go spelling,
which makes the suite implementation-independent and gates all later
backend-rewrite phases.

## Functional Requirements

### FR-1: Behavioral compile runner
The corpus runner SHALL, given a transpile case and a transpiler, transpile the
case input and write the resulting Go into an isolated temporary Go module, then
run `go build` and `go vet` against that module. The source tree SHALL NOT be
modified.

### FR-2: Whole-corpus behavioral test
A test SHALL run every transpile case in the corpus manifest through the
behavioral runner and assert that each case's generated Go builds and vets
cleanly. The test SHALL fail loudly, naming any case whose generated Go fails to
build or vet.

## Acceptance Criteria

- [ ] For every transpile case the runner writes `Output.Go` to a temp module
      and runs `go build` + `go vet` on it.
- [ ] A test asserts every transpile case's generated Go builds and vets
      cleanly.
- [ ] No file under the source tree is created or modified by the runner.
- [ ] Project gates stay green: `go build ./...`, `go vet ./...`,
      `go test ./... -count=1`.

## User Interactions

No user-facing surface. This is internal test infrastructure consumed by the
corpus runner and its tests.

## Error Handling

On a build or vet failure the runner returns a descriptive, case-identified
error including the `go` tool output, so the failing case and reason are
obvious. Read/transpile/setup failures are likewise wrapped and case-identified.

## Out of Scope

- Running doctest sidecars (US-008).
- Package-mode / multi-file fixtures (US-009).
- Changing the exact-match tier or any goldens.

## Open Questions

None.
