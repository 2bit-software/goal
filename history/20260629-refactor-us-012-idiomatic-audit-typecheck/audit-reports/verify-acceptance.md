# Verify: Acceptance Criteria

All criteria met. No CRITICAL. No MAJOR.

- AC-1 (Result/Option/? where natural): no natural conversion exists; the three
  candidate constructs (Load, Check, litClass) are each refused with a concrete,
  behavior-preserving reason recorded in DECISIONS.md. PASS (refusal path).
- AC-2 (goal fix no remaining auto-convertible sites): `goal fix
  selfhost/typecheck/*.goal` produces zero source diff on all five files; only a
  `skipped: [result-sig]` (Load) and advisory `suggestion` (Load, Check) on stderr,
  neither an auto-conversion. PASS.
- AC-3 (depth tests pass against transpiled package): `task check` green, including
  `internal/typecheck` (1.782s) and the selfhost port gate `internal/selfhost`
  (18.376s) that transpiles selfhost/typecheck and runs the copied depth tests. PASS.
- AC-4 (task check/build/fixpoint green): all three green; `task fixpoint` -> FIXPOINT
  OK (selfhost/typecheck/*.go byte-identical across goal-c-1/goal-c-2). PASS.

## Assumptions
- A documented refusal (no source change) is a spec-compliant outcome for an
  idiomatic-audit story, consistent with US-008 and US-010.
