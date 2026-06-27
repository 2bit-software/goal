# Plan Coverage Audit — US-008

Every spec requirement maps to a plan element:

- FR-1 (execute each doctest case) → `RunDoctestExec` writes both files + runs
  `go test`.
- FR-2 (behavioral assertion) → `TestDoctestExecRunner` per-case `t.Run`.
- FR-3 (non-destructive) → temp module + `defer os.RemoveAll`.
- FR-4 (loud empty guard) → `t.Fatalf` when zero ran.

No scope creep (no files modified beyond the two new ones). No CRITICAL/MAJOR
findings.

## Assumptions
- The 4 manifest `kind: doctest` cases are the complete doctest-bearing set.
