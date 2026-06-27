# Tasks Audit

## Coverage
- FR-1 → Task 2; FR-2 → Tasks 1+3; FR-3 → Task 4; FR-4 → Task 4; FR-5 → mirrored in
  Tasks 2/3/4; all AC → Task 5. Complete.
- Every plan file appears in a task (sema.go/resolve.go → T1; mustuse.go → T2;
  implements.go → T3; question.go → T4; check.go + corpus test → T5; unit tests with
  their checks). No scope creep.

## Ordering
- DAG valid: T1 (no dep), T2 (no dep), T4 (no dep), T3 (dep T1), T5 (dep T2,T3,T4).
- Each task leaves the tree compiling: T1 adds nil-safe map fields; T2/T3/T4 add new
  files with self-contained functions not yet called; T5 wires them in and adds tests.
  Building after T1–T4 individually compiles (new funcs are just unused-but-exported,
  which Go permits at package scope).

## Executability
- Each task ≤ 2 files (+ its test), single-turn sized, concrete verify command.
- Interface contract uniform: `func(*ast.File, *Info) []Diagnostic`.

No CRITICAL/MAJOR findings. Approved to implement.
