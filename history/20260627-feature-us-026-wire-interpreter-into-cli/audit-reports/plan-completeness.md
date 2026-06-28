# Plan Audit: Coverage — US-026

Every FR and acceptance criterion maps to a plan element:

- FR-1 (interpreter engine selection) → `--engine=interp` parsing + `cmdRunInterp`.
- FR-2 (success exit) → `cmdRunInterp` returns nil on `Run()` success.
- FR-3 (program output) → `interp.WithStdout(out)`.
- FR-4 (default unchanged) → engine defaults to `ast`; existing `cmdRun` path untouched.
- FR-5 (loud failure) → parse/sema-gate/no-main/runtime errors propagate as the command error.

Acceptance criteria all have a test in the Test Strategy section (happy path,
unknown-engine, no-main). No scope creep: changes confined to cmd/goal.

No CRITICAL or MAJOR findings.

## Assumptions
- Single-file `.goal` input for the interp path; directory runs stay on transpile.
- Full authority (GrantAll) for CLI runs.
