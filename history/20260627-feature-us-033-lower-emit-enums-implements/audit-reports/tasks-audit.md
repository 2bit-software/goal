# Tasks Audit

## Coverage
- AC1 -> Tasks 1-4; AC2 -> Tasks 3,4,5. All requirements covered.
- Plan file inventory (lower.go, backend.go, emit.go, backend_test.go) all
  appear in tasks. No file outside the plan referenced.

## Ordering
- DAG valid: 1 -> 2 -> {3,4} -> 5 -> 6. No cycles, no backward deps.
- Compiles after each task: Task 1 adds standalone helpers (may be briefly
  unused — acceptable mid-feature; Task 3/4 consume them in the same feature
  before verify). Task 2 wiring keeps plain-Go path green. Final gate (Task 6)
  runs the full suite.

## Executability
- Each task names exact files + a concrete verify command. Reference encodings
  point at in-repo known-good source. No placeholders.

Note: Go flags unused package-level funcs only if unused across the package at
build of a consuming target; within one feature 1->3/4 wire them before the
verify gate, so no dead-code build break. Ready to implement.
