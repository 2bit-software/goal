# Tasks Buildability Audit — US-036

Each task is small, ordered, and verifiable:
- T1-T5 are localized edits to `internal/backend/emit.go`.
- T6-T7 are corpus data + one test-line edit.
- T8 adds one test; T9 is the verify gate.

The list is sufficient to implement the spec end-to-end with no guessing. No
CRITICAL/MAJOR findings.

## Assumptions
- `corpus.RunCompile` and `backend.Transpile` signatures are unchanged from
  US-035 (confirmed by reading emit.go/backend_test.go during scoping).
