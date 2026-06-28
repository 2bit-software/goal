# Verification — US-028

## Acceptance Criteria

- [x] A `TypeChecker` interface is defined and the existing typecheck (transpile
  then go/types) implements it — `internal/typecheck/checker.go`:
  `TypeChecker` + `GoTypesChecker` (compile-time assertion
  `var _ TypeChecker = GoTypesChecker{}` in `checker_test.go`).
- [x] A test exercises the depth checks through the interface and the existing
  typecheck cases still pass — `TestTypeCheckerInterfaceParity` drives clean +
  diagnostic-producing packages through the interface value and asserts parity
  with the concrete `Load`+`Check*` path; `TestTypeCheckerErrorsOnBadTranspile`
  covers the error path. All existing `internal/typecheck/*_test.go` remain green.
- [x] The CLI depth-check entry point resolves diagnostics through the interface
  value — `cmd/goal/main.go` `runDepthChecks` now uses
  `var tc typecheck.TypeChecker = typecheck.GoTypesChecker{}`.

## Verify Gates (prd.json verifyCommands)

- [x] `go build ./...` — clean
- [x] `go vet ./...` — clean
- [x] `go test ./... -count=1` — all packages ok (incl. typecheck, cmd/goal,
  and the full corpus suite)

## Assumptions

- The seam returns `[]Diagnostic` (not `*Package`) so a future native checker —
  which produces no go/types view — can satisfy it; chosen deliberately over a
  `Load`-returning interface.
- Depth-check order (implements, must-use, no-zero-value) preserved from the
  prior inline caller, so diagnostic ordering is unchanged.
- No behavior change to depth-check logic, messages, codes, or positions
  (pure structural seam).
