# Interface-based Transpile Runner — Business Specification

## Overview

The golden transpile corpus must be executable against any goal front-end that
produces transpile output, not only the current pipeline. This feature provides
a transpiler-agnostic runner: given a corpus case and a transpiler, it judges
whether the transpiler produces the expected Go. The comparison is format
insensitive so that two pieces of equivalent Go that differ only in whitespace
still match.

## Functional Requirements

### FR-1: Pluggable transpiler seam
The runner SHALL accept any transpiler that lowers goal source to the standard
transpile output (lowered Go plus an optional doctest sidecar). The current
pipeline front-end SHALL be usable as such a transpiler without modification.

### FR-2: Format-insensitive comparison
The runner SHALL gofmt-normalize both the produced output and the golden before
comparing them, so equivalent Go differing only in formatting compares equal.

### FR-3: Whole-corpus execution
Every transpile case in the corpus manifest SHALL pass when executed against the
current pipeline front-end.

### FR-4: Doctest-sidecar tolerance
Some manifest transpile cases carry a doctest `_test.go` sidecar as their golden
rather than the main Go output. The runner SHALL treat such a case as passing
when the golden matches the produced output — either the main Go or, when
present, the doctest sidecar.

## Acceptance Criteria

- [ ] A transpiler interface exposing a single transpile operation exists, and
      the current pipeline transpile function can be supplied through it.
- [ ] Running a transpile case produces a pass when the gofmt-normalized golden
      equals the gofmt-normalized produced Go.
- [ ] Running a feature-11 doctest example (whose golden is a `_test.go` sidecar)
      produces a pass.
- [ ] A test runs every transpile case in the manifest against the pipeline and
      all pass; the test fails loudly if no transpile cases are found.
- [ ] A mismatch, a missing input/golden file, or a transpile failure produces a
      descriptive, case-identified error.

## User Interactions

This is an internal test/runner API consumed by the corpus test suite. There is
no end-user CLI or UI surface.

## Error Handling

- A read failure (input or golden missing) yields an error naming the case.
- A transpile failure yields an error naming the case and wrapping the cause.
- An output mismatch yields an error naming the case and showing got vs want.

## Out of Scope

- The check runner (inline `// want` markers) — a later story.
- A dedicated doctest sidecar runner / behavioral (compile/run) tiers — later
  stories.
- Rewiring the existing pipeline/check test harnesses to delegate to this runner
  — a later story.
- Re-classifying the manifest's doctest cases; their existing classification is
  preserved.

## Open Questions

- None. The doctest-sidecar tolerance (FR-4) resolves the only ambiguity, which
  was how feature-11 cases (sidecar goldens) coexist with ordinary transpile
  cases in one runner.
