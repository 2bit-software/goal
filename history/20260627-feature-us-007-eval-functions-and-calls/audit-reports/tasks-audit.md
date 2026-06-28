# Tasks Audit

## Findings

No CRITICAL or MAJOR. The four tasks are ordered by dependency (value model ->
registration/signal -> call+control-flow -> tests), each keeps the package
compiling, and each touches <=2 files. Every FR maps to a task and every file in
the plan inventory appears. Verification steps are concrete (named go test runs /
build / vet).

- MINOR: Tasks 2 and 3 both edit interp.go; they remain independently committable
  because Task 2 leaves a compiling package (Run still works on main's body) and
  Task 3 layers calls on top. Acceptable.

## Assumptions
- A single loop iteration implements all four tasks together and commits once
  (loop-runner contract), so per-task commits are not required — the ordering
  exists to keep the build green incrementally during implementation.
