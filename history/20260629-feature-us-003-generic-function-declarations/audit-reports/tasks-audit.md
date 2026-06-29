# Tasks Audit

## Coverage
- All FRs covered: FR-1 (T1,T2), FR-2 (T2,T4), FR-3 (T3,T4), FR-4 (T1).
- All plan files appear across tasks (ast.go, walk.go, parser.go, emit.go, test).
- No scope creep.

## Ordering
- Valid DAG: T1 -> {T2, T3} -> T4. Compiles after each (field added first).

## Executability
- Each task has concrete instructions referencing exact functions/lines and a
  runnable verify command.
- Each task touches <= 2 files.

## Sizing
- Well sized; small but each is a distinct, meaningful change.

No CRITICAL/MAJOR findings.

## Assumptions
- Tasks 2 and 3 are independent (both depend only on T1) and can be done in one
  agent turn together since the whole change is small.
- The test in T4 lives in an existing backend test file following its style.
