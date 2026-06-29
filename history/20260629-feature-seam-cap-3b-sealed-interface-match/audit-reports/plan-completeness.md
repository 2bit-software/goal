# Plan Coverage Audit — SEAM-CAP-3b

Every FR and acceptance criterion maps to a plan element:
- FR-1 registry -> sema.go/resolve.go SealedImpls.
- FR-2 parser -> goal_match.go + ast TypePattern.
- FR-3 lowering -> backend emit.go sealedMatch + dispatch.
- FR-4 exhaustiveness -> check.go.
- FR-5 mirror -> selfhost/* row set.
- AC behavioral/sema -> the two new test files + fixtures.
- AC gates -> task check/build/fixpoint.

No CRITICAL/MAJOR. No scope creep (cross-package explicitly excluded).

## MINOR
- The behavioral test build mechanism reuses an existing pattern
  (crosspkg_goal_enum_test); confirm whether single-file `backend.Transpile` or
  package-mode is needed. Same-package single-file suffices, so `backend.Transpile`
  (single file) is the simplest harness.

## Assumptions
- Single-file fixture is sufficient for "same package" proof; multi-file union is
  implemented (Merge) but the proof need not exercise multiple files.
