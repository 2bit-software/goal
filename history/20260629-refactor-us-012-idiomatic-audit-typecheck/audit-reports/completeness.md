# Audit 1: Completeness

No CRITICAL findings. No MAJOR findings.

## MINOR
- The spec correctly anticipates a likely all-refusal outcome but keeps the conversion
  requirement (FR-1) conditional ("or the non-applicability is recorded as a refusal"),
  which is the right framing for an audit story whose outcome depends on the package survey.

## Coverage
- Happy path (idiomatic conversion where it fits): FR-1.
- Machine gate (no auto-convertible sites): FR-2 / AC-2.
- Behavioral equivalence + fixpoint: FR-4 / AC-3, AC-4.
- Error-handling preservation explicitly stated (Load wrapping, Check surfacing).
- Out-of-scope is specific (cross-package switch->match, oracle-pinned signatures, ordered
  iota ints).

## Assumptions
- The US-003 verbatim selfhost is the behavioral oracle and its pinned signatures
  (`Load`, `TypeChecker.Check`) must not change.
- A documented refusal (no source change) is an acceptable, spec-compliant outcome for an
  idiomatic-audit story, consistent with US-008 and US-010.
