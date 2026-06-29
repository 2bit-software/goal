# Tasks Audit

## Coverage
- Every FR/AC maps to a task (T1/T2 capability+mirror, T3 sema exhaustiveness, T4 behavioral,
  T5 gates+DECISIONS). Every plan file appears in a task. No scope creep.

## Ordering
- Valid DAG: T1 → {T2, T3, T4} → T5. No circular refs. Tree compiles after T1 (additive
  return + merge); T2 keeps the port gate green; tests in T3/T4 depend only on T1.
- NOTE: the PostToolUse `task check` hook runs the whole suite after each edit, so transient
  redness between T1 and T2 (selfhost mirror not yet updated) is expected and clears at T2 —
  consistent with the documented multi-file-refactor pattern in progress.txt.

## Executability
- Each task has concrete instructions referencing exact functions and the
  crosspkg_goal_enum_test.go precedent, and a runnable verify command. Each touches ≤ 5 files.

## Sizing
- Tasks are single-turn sized; none trivial. T1 is the core; T2 is a mechanical mirror.

## Findings
- No CRITICAL/MAJOR/MINOR blocking issues.

## Assumptions
- Treating T1+T2 as separate tasks despite the line-for-line mirror requirement, because the
  hook tolerates transient redness; they will be committed together as one story.
