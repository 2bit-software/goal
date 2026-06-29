# Plan coverage audit

Every spec requirement traces to a plan element:
- FR-1 (marker satisfaction) → backend implementsMarker + sealedEmbeds.
- FR-2 (registration cascade) → sema cascadeSealedImpls.
- FR-3 (exhaustiveness both levels) → existing checkOneSealedMatch over cascaded
  SealedImpls; sema test.
- FR-4 (behavior parity) → backend nested_sealed_test go test vs reference switch.
- FR-5 (transitivity) → cascade + sealedEmbeds both transitive.
- AC fixpoint/corpus → verifyCommands.

No scope creep. No CRITICAL/MAJOR findings.

## Assumptions
- Cascade idempotent + run in both Resolve and ResolvePackage.
- No foreign.go change needed (cascade rides existing projection).
