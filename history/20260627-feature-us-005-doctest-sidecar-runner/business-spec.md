# Doctest Sidecar Runner — Business Specification

## Overview

The golden corpus is indexed by a runner-independent manifest so any front-end
can be judged against the same cases. Transpile and check cases already have
runners. Doctest cases — goal sources whose golden output is an emitted
`_test.go` doctest sidecar — are not yet a first-class part of the suite. This
feature adds doctest cases to the manifest and a runner that compares the
transpiler's emitted doctest sidecar against the golden, so doctest output is
verified as part of the same corpus suite.

## Functional Requirements

### FR-1: Doctest cases exist in the manifest
The generated manifest SHALL include a doctest case for every goal source whose
golden output is a doctest sidecar (an emitted `_test.go`). The pre-existing
transpile and check case counts SHALL remain unchanged.

### FR-2: Doctest runner compares the sidecar
The runner SHALL transpile a doctest case's input and compare the transpiler's
emitted doctest sidecar output against the case's golden, treating two sidecars
that differ only in formatting as equal.

### FR-3: Whole-corpus doctest test
A test SHALL run every doctest case in the committed manifest against the
current transpiler and assert all pass. It SHALL fail loudly if the manifest
yields zero doctest cases, so an empty or mis-generated manifest cannot
masquerade as green.

## Acceptance Criteria

- [ ] The runner compares the emitted doctest sidecar to the golden sidecar,
      normalized so formatting-only differences do not fail a case.
- [ ] Every doctest case in the manifest passes against the current transpiler.
- [ ] The doctest test fails loudly when no doctest cases are present.
- [ ] The previously asserted corpus counts (51 transpile pairs, 50 check
      cases) remain true after the change.
- [ ] `go build ./...`, `go vet ./...`, and `go test ./... -count=1` are green.

## User Interactions

No end-user interface. The audience is the compiler test suite: maintainers run
`go test ./internal/corpus/...` and the new doctest tier participates.

## Error Handling

The runner returns a descriptive, case-identified error on any input read
failure, transpile failure, normalization failure, or sidecar mismatch, and nil
when the case passes. The whole-corpus test surfaces each failing case by ID.

## Out of Scope

- Executing doctests (running `go test` on the emitted sidecar) — that is a
  later behavioral-tier story.
- Reclassifying or moving any existing golden source file.
- Changing transpiler behavior.

## Open Questions

- None. Doctest cases are identified by golden content (an emitted `_test.go`
  sidecar), which deterministically selects exactly the doctest-bearing
  examples.
