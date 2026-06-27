# Audit — AI-Consumer Readiness

- Terms defined: struct literal, variant construction, spread opt-out, deferral —
  all mapped to concrete AST node types in research.md.
- Data formats specified: diagnostic Severity (Error/Warning), Code
  (`missing-field`/`unresolved-literal-type`), and exact Message templates given.
- State transitions explicit: omission→Error, unresolved-type→Warning,
  spread/complete/data-less→no diagnostic, match-pattern→ignored.
- Acceptance criteria are test-assertable: each maps to a named file in
  testdata/check/08-no-zero-value with an inline `// want` marker (or a no-marker
  clean case where any Error fails).

An AI agent can implement this without clarifying questions. No CRITICAL/MAJOR.

## Assumptions

- The corpus runner (corpus.RunCheck) fails a case only on an unclaimed
  Error-severity diagnostic; unclaimed Warnings are allowed — so deferral
  Warnings on clean lines do not fail their case.
- sema.Check aggregates the new CheckFields alongside CheckExhaustive; the existing
  corpus.SemaCheck adapter forwards all sema diagnostics with no change.
