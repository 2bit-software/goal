# Corpus Interpreter Runner (US-025) — Business Specification

## Overview

The goal project keeps a behavioral conformance tier that judges a back-end by
the OBSERVABLE behavior of the programs it runs, not by the Go it spells. That
tier was designed to be implementation-independent — it is the same conformance
suite the Go transpiler and the goscript interpreter both run
(REWRITE-ARCHITECTURE.md §6). The Go back-end is already judged this way
(`corpus.RunDoctestExec`). This feature gives the goscript interpreter the same
treatment: a corpus runner that loads a corpus case, runs it through the
interpreter, and asserts the observable behavior matches the case's expected
result — reusing the existing corpus model, never inventing a new oracle.

## Functional Requirements

### FR-1: Interpreter corpus runner
The corpus package SHALL provide a runner (`RunInterp`) that takes a corpus case
and runs its source through the goscript interpreter, comparing observable
behavior the same way the Go behavioral doctest tier does.

### FR-2: Doctest observable-behavior comparison
For a doctest case, the runner SHALL evaluate each documented `>>>` example
expression through the interpreter and compare the rendered result against the
example's expected output line(s). A match passes; a mismatch fails.

### FR-3: Loud, case-identified failures
A doctest whose evaluated result differs from its expected rendering, or a case
that cannot be evaluated (parse/eval failure), SHALL produce a descriptive,
case-identified error that names the offending case, the input expression, the
expected value, and the actual value. A failure SHALL NEVER be reported as a
pass.

### FR-4: Wrong-kind / empty-case refusal
The runner SHALL refuse a case of the wrong kind with a descriptive error, and a
doctest case that yields no examples SHALL be a loud failure rather than a silent
green.

### FR-5: Behavioral parity oracle
The rendered-result comparison SHALL use the same value spelling the doctest
golden uses (an integer renders as `5`, a string renders quoted as `"abab"`, a
bool as `true`), so a result that would pass the Go doctest tier passes here and
one that would fail there fails here.

## Acceptance Criteria

- [ ] The corpus package exposes `RunInterp`, which loads a corpus case and runs
      it through the interpreter, comparing observable behavior (doctest output /
      asserted result) the way the Go behavioral tier does.
- [ ] Running every doctest corpus case through `RunInterp` passes.
- [ ] A doctest whose expected value is altered to a wrong value makes `RunInterp`
      return a descriptive, case-identified error (no silent pass).
- [ ] A wrong-kind case passed to `RunInterp` returns a descriptive error.
- [ ] The interpreter path introduces no dependency on go/types or the depth
      checker (the native-only envelope holds).

## User Interactions

This is an internal test-infrastructure capability. The interaction is via the
Go test suite: a `corpus` test iterates the manifest's doctest cases and runs
each through `RunInterp`, failing the suite loudly if any case fails or if the
manifest yields no doctest cases.

## Error Handling

- Wrong-kind case → descriptive error naming the case ID and the expected kind.
- Unreadable input file → descriptive, case-identified read error.
- Parse or evaluation failure of a doctest expression → descriptive,
  case-identified error naming the function, the input, and the failure.
- Result mismatch → descriptive error naming the case, function, input, expected,
  and actual values.

## Out of Scope

- Running non-doctest corpus kinds (transpile, check) or package-mode cases
  through the interpreter. This feature covers the doctest (Mode=file) behavioral
  comparison; the whole-corpus interpreter gate is US-027.
- Wiring the interpreter into the CLI (US-026).
- Any new corpus manifest entries or fixtures.

## Open Questions

None — the corpus model, the doctest example structure, and the interpreter
expression-evaluation seam already exist; this feature composes them.
