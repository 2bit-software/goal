# Tasks

## Task 1 — Record the token-kind representation decision in DECISIONS.md
- **Depends on:** none (foundation)
- **Files:** `DECISIONS.md`
- **Spec coverage:** FR-1 (AC-1)
- **Do:** Append a new section under the self-host idiomatic audit recording the
  deliberate decision to keep `selfhost/token`'s `Kind` as an iota-based const
  block rather than a goal `enum`. Rationale: goal `enum` lowers to a sealed
  interface + per-variant struct (DECISIONS.md §01-enums / §8.1), which is not an
  ordered integer type and cannot be array-indexed (`kindNames[k]`) or
  range-compared (`literalBeg < k && k < literalEnd`). Also note: no `switch` to
  convert to `match`; `Lookup` stays comma-ok `(Kind, bool)` (oracle-pinned, not
  a goal-fix site); package is import-free with no `(T,error)` functions.
- **Done when:** DECISIONS.md contains the dated entry in the file's existing
  decision/assumption format.

## Task 2 — Verify gates
- **Depends on:** Task 1
- **Files:** none (verification only)
- **Spec coverage:** FR-2 (AC-2), FR-3 (AC-3/4/5)
- **Do:** Run `goal fix selfhost/token/token.goal` (expect no diff, no report);
  `task check`; `task build`; `task fixpoint`.
- **Done when:** goal fix reports none and all three gates are green.

## Coverage check
- FR-1 → Task 1
- FR-2 → Task 2
- FR-3 → Task 2
- Plan file inventory (`DECISIONS.md`) → Task 1
