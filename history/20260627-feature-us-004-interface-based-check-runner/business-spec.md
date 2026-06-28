# US-004 Interface-Based Check Runner — Business Specification

## Overview
The golden corpus is indexed by a runner-independent manifest. Transpile cases
already run through a Transpiler interface (US-003). This feature adds the
parallel seam for checker cases: a Checker interface and a runner that judges any
checker front-end against the same inline `// want` markers the corpus has always
used. This decouples checker conformance from the existing test harness's
hardcoded paths and lets a future front-end be judged identically.

## Functional Requirements

### FR-1: Checker seam
The corpus model SHALL define a Checker abstraction that, given goal source text,
returns the located diagnostics produced for that source, plus an adapter so the
existing free checker entry point satisfies it without modification.

### FR-2: Marker matching
For a check case, the runner SHALL parse inline `// want "substr"` markers. Each
marker SHALL be satisfied by some diagnostic on the SAME source line whose message
contains the substring. An unsatisfied marker fails the case.

### FR-3: Unexpected-rejection guard
The runner SHALL fail the case on any Error-severity diagnostic that has no marker
on its line (an unclaimed unexpected rejection). Warning-severity diagnostics
(located deferrals) MAY go unclaimed.

### FR-4: Whole-corpus check test
A test SHALL run every check-kind case in the committed manifest through the
checker entry point and all SHALL pass. The test SHALL fail loudly if the manifest
yields zero check cases.

## Acceptance Criteria

- [ ] The corpus model defines a Checker interface and a runner that matches each
      `// want "substr"` marker against a diagnostic on the same line.
- [ ] The runner fails on any unclaimed Error-severity diagnostic.
- [ ] The runner allows unclaimed Warning-severity diagnostics.
- [ ] A test runs every check case in the manifest against the checker entry point
      and all pass.
- [ ] The test fails loudly when no check cases are present.
- [ ] go build ./..., go vet ./..., and go test ./... -count=1 are green.

## User Interactions
None directly user-facing. Consumed by the corpus test suite and future runners.

## Error Handling
- Read failure on a case input: descriptive, case-identified error.
- Checker internal error: surfaced as a case-identified error.
- Unsatisfied marker / unclaimed Error: case fails with the line and substring or
  the offending diagnostic identified.

## Out of Scope
- Package-mode check cases (single-file file-mode only here).
- Rewiring the existing check_test.go harness to delegate to this runner (US-006).
- Doctest and behavioral tiers (later stories).

## Open Questions
None — the pattern is fully established by US-003 and the existing check harness.
