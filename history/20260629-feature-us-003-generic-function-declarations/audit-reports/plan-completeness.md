# Plan Coverage Audit

- FR-1 (parse) -> parser.go change. Covered.
- FR-2 (constrained params) -> reuses parseTypeParams (handles constraints). Covered.
- FR-3 (faithful transpile) -> emit.go funcSig change. Covered.
- FR-4 (non-generic unchanged) -> field is nil-guarded; full suite. Covered.

No scope creep. No CRITICAL/MAJOR findings.

## Assumptions

- No sema change is needed; sema treats generic funcs' params/results as
  ordinary type expressions and the type-param identifiers as in-scope. To be
  verified at implement time by running the full suite + go build.
