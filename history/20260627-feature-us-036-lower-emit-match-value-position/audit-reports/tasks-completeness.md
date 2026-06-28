# Tasks Completeness Audit — US-036

All plan components are covered by a task, in dependency order:
- Lowering helper + field export (T1, T2) precede dispatch wiring (T3-T5).
- Corpus fixture (T6) precedes the count-test fix (T7) and the behavioral test
  (T8), all before the verify gate (T9).

No CRITICAL or MAJOR findings. Each task is independently checkable and maps to a
named seam. The count-test update (T7) and splice-generated golden (T6) — the two
MAJOR risks from plan-audit — are explicit tasks.

## Assumptions
- The fixture lives in the feature-02 examples dir so `Generate` indexes it as a
  file-mode transpile case (hence the 51->52 bump).
