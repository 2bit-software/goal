# Plan Coverage Audit — US-004

Every spec requirement traces to a plan element:

- FR-1 / AC-1 (autofix run + committed) -> step 2 (`goal fix --inplace selfhost`) + commit.
- FR-2 (never emit non-compiling code) -> `internal/fix/resultsig.go` changes.
- FR-3 / AC-2 (fixed point) -> step 2 second-run idempotence check.
- FR-4 / AC-3..5 (gates green) -> step 3 (`task check/build/fixpoint`).
- AC-6 (existing fix tests) -> Testing Strategy (`go test ./internal/fix ./cmd/goal`).

No scope creep: the only code change is to the fixer; selfhost changes are
whatever the corrected fixer produces.

No CRITICAL or MAJOR findings.

## Assumptions
- The corrected fixer produces zero selfhost source changes (validated empirically
  in the next step via dry-run reasoning); if it produces changes, the gates are
  the backstop.
