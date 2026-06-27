# Tasks Audit

- Every spec FR maps to a task (FR-1: T1/T3, FR-2: T2/T3, FR-3: T2, FR-4: T1/T2).
- Every plan file maps to a task (protocol.go T1, semantictokens.go T2,
  server.go T3, tests T4).
- Ordering is a valid topological sort: T1 (no deps) -> T2 -> T3 -> T4.
- No task touches more than 5 files.

No CRITICAL/MAJOR findings. Ready to implement.
