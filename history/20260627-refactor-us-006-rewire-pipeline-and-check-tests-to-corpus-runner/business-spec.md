# US-006 Rewire existing harnesses to runner — Business Specification

## Overview

The golden corpus is now indexed by a single manifest (`corpus/manifest.json`)
and exercised by runner functions in `internal/corpus`. Two legacy test
harnesses still re-derive case paths by hand: the pipeline harness hardcodes a
list of per-feature example directories and the testdata directory, and the
checker harness walks a hardcoded `testdata/check` root. This story retires those
hardcoded paths by having both harnesses consume the corpus manifest through the
shared runners, so there is exactly one source of truth for which cases exist.

## Functional Requirements

### FR-1: Pipeline harness consumes the manifest
The pipeline test harness SHALL obtain its transpile and doctest cases from the
corpus manifest and exercise each through the shared corpus runner, rather than
from any hand-maintained list of feature/example directories.

### FR-2: Checker harness consumes the manifest
The checker test harness SHALL obtain its check cases from the corpus manifest
and exercise each through the shared corpus check runner, rather than by walking
a hardcoded `testdata/check` directory.

### FR-3: No regression in coverage
Every case the legacy harnesses exercised — testdata transpile pairs, per-feature
regression examples, feature-11 doctests, and `testdata/check` cases — SHALL
continue to be exercised after the rewrite.

### FR-4: Fail loudly on an empty corpus
Each rewired harness SHALL fail (not silently pass) if the manifest yields zero
cases of the kind it drives, so a mis-generated or empty manifest cannot
masquerade as green.

## Acceptance Criteria

- [ ] `internal/pipeline/pipeline_test.go` contains no hardcoded feature-dir list
      and no `filepath.Join("..","..","features",...)` path (grep returns nothing).
- [ ] `internal/check/check_test.go` contains no hardcoded `testdata/check` walk
      and no hardcoded feature paths (grep returns nothing).
- [ ] `go build ./...` succeeds.
- [ ] `go vet ./...` is clean.
- [ ] `go test ./... -count=1` is green.
- [ ] The pipeline harness exercises every transpile and doctest case from the
      manifest; the checker harness exercises every check case from the manifest.
- [ ] Each rewired harness fails loudly when its case count is zero.

## User Interactions

Developer-facing only. Running `go test ./internal/pipeline/...` and
`go test ./internal/check/...` exercises the corpus through the shared runners.
Adding or removing a golden case is done by regenerating the manifest, not by
editing these test files.

## Error Handling

- A missing or unreadable manifest fails the test with a clear, path-identified
  error.
- A zero-case result for a driven kind fails the test loudly (FR-4).
- Individual case failures are reported per-case via subtests with the case ID.

## Out of Scope

- Changing the runner semantics or the manifest format (US-001..US-005 own those).
- Rewiring other harnesses (foreign_test.go, package-mode tests, per-check
  unit tests like exhaustive_test.go) — only `pipeline_test.go` and
  `check_test.go` are in scope.
- Adding behavioral (compile/run) tiers — that is US-007/US-008.

## Open Questions

- None. The runners, manifest, and adapters already exist; this story only
  redirects the two named harnesses onto them.
