# Audit: Completeness — US-021

## Findings

No CRITICAL findings. No MAJOR findings.

- MINOR: The spec defers pointer/Option field recursion (Out of Scope). The
  file-mode features/12 fixtures used for the unit test do not require it, so this
  does not block the acceptance criteria. The deferral is explicit and refused
  loudly, satisfying FR-5.
- MINOR: Map iteration ordering is not user-visible for a struct-producing derive
  (a converted map is compared by entries, not order), so no determinism clause is
  needed here.

## Assumptions

- The canonical "12-derive-convert shape" for the acceptance test is
  `derive_nested_struct.goal` (identity + registry bridge + nested struct), which
  exercises the core of FR-1/FR-2 in one program.
- `from func` conversions are invoked by their registered name through the root
  scope (the established `callConversion` pattern), so no new callable mechanism
  is introduced.
