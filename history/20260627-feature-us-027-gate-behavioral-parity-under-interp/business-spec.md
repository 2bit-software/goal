# US-027 Gate: behavioral parity under interp — Business Specification

## Overview

The goal corpus is the implementation-independent behavioral conformance tier:
the one yardstick every back-end is measured against. The Go (AST) back-end is
already proven across the whole corpus by a behavioral gate. This feature adds
the parallel gate for the goscript interpreter, so the whole applicable corpus
is proven to behave identically under interpretation — the interpreter is held
to the same standard as the Go back-end.

The interpreter participates in the behavioral tier through the doctest corpus
subset (each `///  >>>` example's evaluated runtime result must equal its
documented expected value), run in-process with no Go toolchain via
corpus.RunInterp. "Every applicable corpus case" therefore means every
doctest-kind case.

## Functional Requirements

### FR-1: Whole-corpus interpreter behavioral gate
A single gate test SHALL run every applicable (doctest-kind) corpus case in the
committed manifest through the interpreter and SHALL report a failure for any
case whose observable behavior does not match its documented expected value.

### FR-2: Explicit, justified skip list
Any case excluded from the gate SHALL be enumerated in an explicit skip list
that maps the case ID to a human-readable justification (for example, "uses an
unshimmed stdlib symbol"). No applicable case may be silently dropped.

### FR-3: Gate fails on behavioral failure
The gate SHALL fail if any non-skipped applicable case fails behaviorally under
interpretation.

### FR-4: Gate fails on an unjustified skip
The gate SHALL fail if any skip-list entry lacks a recorded (non-blank) reason.

### FR-5: Loud on an empty/narrowed corpus
The gate SHALL fail if the manifest is empty or yields zero applicable cases, so
a mis-generated or filtered manifest cannot masquerade as green. A skip-list
entry that does not name a real applicable case in the manifest SHALL fail the
gate (no stale/orphan skips).

## Acceptance Criteria

- [ ] A gate test runs every applicable corpus case through the interpreter via
      RunInterp with zero behavioral failures.
- [ ] Every excluded case appears in an explicit skip list with a justification;
      none is silently dropped.
- [ ] The gate fails if a case fails behaviorally.
- [ ] The gate fails if the skip list is missing a recorded reason for any entry.
- [ ] The gate fails loudly if zero applicable cases ran.
- [ ] The project verifyCommands (go build, go vet, go test ./...) stay green.

## User Interactions

Developer-facing only: `go test ./internal/corpus` runs the gate. There is no
end-user-visible surface.

## Error Handling

- A behavioral mismatch surfaces as a case-identified test failure (RunInterp's
  descriptive, case-named error).
- An unjustified or stale skip-list entry surfaces as a named test error
  identifying the offending case ID.
- An empty/narrowed manifest surfaces as a fatal test error.

## Out of Scope

- Running non-behavioral tiers (transpile/check/package) through the
  interpreter — the interpreter's behavioral tier is the doctest subset.
- Adding or modifying corpus cases.
- The script-to-module no-op upgrade gate (US-028).

## Open Questions

- None. All seams (RunInterp, Load, the manifest) exist; the current corpus has
  four doctest cases and all pass under interpretation, so the skip list ships
  empty but the enforcement mechanism is exercised by a focused unit test.
