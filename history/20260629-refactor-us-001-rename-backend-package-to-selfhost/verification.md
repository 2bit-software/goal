# Verification — US-001

## Verify commands (prd.json verifyCommands)
- `task check` — PASS (go vet + full `go test ./...`, all packages ok)
- `task build` — PASS (bin/goal, bin/goalc)
- `task fixpoint` — PASS (`FIXPOINT OK`; selfhost/backend/*.go now emitted
  byte-identically across both bootstrap stages)

## Story-specific
- `go test ./internal/selfhost -run TestPortedBackendPackage` — PASS (compile gate
  BuildTranspiled + behavioral gate BuildAndTest over the self-contained suite)

## Acceptance criteria
- AC-1: selfhost/backend holds the 6 non-test files as verbatim .goal copies — MET
  (diff against internal/backend/*.go empty; zero reserved-word renames needed).
- AC-2: transpile-and-build smoke gate compiles backend + ported deps — MET.
- AC-3: existing backend tests (minus fixture-dependent ones) pass against the
  transpiled package — MET (12 self-contained tests via backend_selfhost_test.go).
- AC-4: task check / build / fixpoint green — MET.
