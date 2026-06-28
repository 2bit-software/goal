# Tasks Audit — US-023

Tasks: tasks.md

## Coverage
- FR-1 → Task 1 (gate) + Task 2 (route the effect). Covered.
- FR-2 → Task 1 (GrantAll default) + Task 3 (assert). Covered.
- FR-3 → Task 1 (sink field/option) + Task 3 (capture). Covered.
- Every plan file appears: interp.go (Task 1), host.go (Task 2),
  cap_io_test.go (Task 3).

## Ordering
- Valid topological order: Task 1 (fields/gate) → Task 2 (route through gate) →
  Task 3 (tests). No forward references.

## Executability
- Each task names concrete files, imports, signatures, and a concrete verify
  command. Task 2's interception point (`evalHostCall`, branch on `key`) is
  specific.

## Sizing
- Each task touches a single file (≤ 3-5), independently committable: after
  Task 1 the package still builds (new field/gate unused-but-valid), after
  Task 2 the routing compiles, after Task 3 the suite is green.

## Findings
- No CRITICAL/MAJOR. MINOR: Task 2 must drop the `os` import from host.go if it
  becomes unused, or `go vet`/build fails — already called out in the task.

## Assumptions
- The existing host_test.go `newInterp` helper is reusable for the program-parse
  path; the sink test constructs `New(...)` directly to inject `WithStdout`.
