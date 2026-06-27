# Verify: Quality — US-021

## Checks
- Error handling: unsourced field, unconvertible pair, and pointer/Option
  recursion all return descriptive, located refusals (never a silent zero) —
  satisfying FR-5. Fallible conversion errors are threaded via the deriveErrVal
  signal and surfaced as the derive's `(T, error)` second result.
- The tests assert real behavior (struct TypeIDs, field values, error message
  text), not just non-nil — they would fail if the conversion were wrong.
- Dependency envelope preserved: type-string splitters are local to interp; no
  internal/backend import.

## Findings
- MINOR: Pointer/Option field recursion is deferred (loud refusal), per the spec's
  Out of Scope. The file-mode features/12 fixtures used for testing do not
  exercise it; cross-package derive coverage stays on the package-mode corpus
  runner.
- MINOR: Bodied-override evaluation (FR-4) is implemented (deriveOverridesOf +
  override eval) but not separately unit-tested; the nested-struct and fallible
  tests cover the implicit-fill path, and the override path mirrors the proven
  backend logic. Not blocking.

No CRITICAL/MAJOR findings.

## Assumptions
- The canonical acceptance shape is `derive_nested_struct`; fallible + refusal
  tests are added beyond the single required test for robustness.
