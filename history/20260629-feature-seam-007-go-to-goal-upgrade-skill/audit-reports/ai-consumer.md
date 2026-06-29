# Audit: AI-Consumer Readiness

## CRITICAL
None.

## MAJOR
None. The research summary pins exact goal syntax (enum/match/sealed/implements/
Result/?), the `goal fix` I/O contract (stdout vs stderr, `-inplace`), the
reserved-word list, and the idiom-catalogue decision table — an agent can
implement without guessing.

## MINOR
- MINOR: "DECISIONS-style summary" should be given a concrete shape. Resolution:
  skill specifies a fixed report template (Converted / Refused-with-reason /
  Build status) so the output is reproducible.
- MINOR: acceptance criterion "dogfooded on at least one real example" needs a
  named example for test assertions. Resolution: skill ships a documented example
  under references/ with the exact commands and before/after.

## Assumptions
- `goal` binary is on PATH (built via `task build` -> `bin/goal`, or `task
  install`). The skill states this as a prerequisite.
- Verification of "still builds" uses `goal build <scope-dir>` for a package and a
  temp single-file package for a lone file.
