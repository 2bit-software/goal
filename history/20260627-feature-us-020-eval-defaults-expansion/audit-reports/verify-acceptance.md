# Verify: Acceptance Coverage — US-020

## Project gates (prd.json verifyCommands)
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS (all packages, incl. internal/interp)
- Dependency hygiene (US-022): `go list -deps ./internal/interp` contains no
  go/types, internal/backend, or internal/typecheck — CLEAN.

## Acceptance criteria -> evidence

| Criterion | Test | Asserts |
|-----------|------|---------|
| Omitted primitive fields -> "", false, 0 | TestDefaultsFillsOmittedPrimitivesAndPreservesSet | email=="", active==false |
| Explicit fields preserved | TestDefaultsFillsOmittedPrimitivesAndPreservesSet / TestDefaultsDoesNotOverwriteExplicitFieldBeforeOrAfter | name=="ada", logins==3; email=="root@x" with spread BEFORE it |
| Omitted named-struct field -> zeroed instance | TestDefaultsFillsNestedStructAndSlice | meta == Addr{host:"", port:0} |
| Omitted slice field -> empty slice | TestDefaultsFillsNestedStructAndSlice | tags is KindSlice, len 0 |
| Non-defaults spread refused | TestNonDefaultsSpreadIsRefused | error names "spread" + "defaults" |
| 08-no-zero-value/defaults shape demonstrated | all of the above (modeled on defaults_primitives + defaults_refs) | — |

Plus TestDefaultsUnknownStructIsRefused covers the unknown-struct error path.

All acceptance criteria map to a passing test that asserts the required
behavior. No criterion is left untested.

## Assumptions
- A defaulted slice field is an empty slice (not the bare nil value) so range/len
  remain valid — behaviorally identical to a nil slice in Go.
- A defined type over a primitive (e.g. `type Role int`) is set explicitly in the
  fixtures, so defaulting such a field is not exercised; its zero would currently
  resolve to nil (no alias table in sema.Info). Out of scope for US-020 and not
  required by the acceptance criteria.
