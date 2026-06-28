# Tasks — US-022 Gate interp on native sema only

## T1: Add the native sema gate to Run()
- In `internal/interp/interp.go`, add a `gate()` method that runs
  `sema.Check(ip.file, ip.info)` and returns a located, named error for the
  first `sema.Error`-severity diagnostic (Pos.String() + Code + Message), nil
  otherwise (Warnings skipped).
- Call `ip.gate()` as the first statement of `Run()`, returning its error
  before `findMain`.
- Document the seam (REWRITE-ARCHITECTURE.md §3.2: native checks only).

## T2: Acceptance + dependency tests
- New `internal/interp/gate_test.go` (package interp, stdlib testing):
  - `TestRunRefusesNonExhaustiveMatch` — non-exhaustive match in main is
    refused with a located error (code + line:col).
  - `TestRunAllowsExhaustiveMatch` — clean program runs (nil).
  - `TestRunDoesNotBlockOnWarning` — unresolved-enum warning does not refuse.
  - `TestInterpHasNoGoTypesOrTypecheckDep` — `go list -deps ./internal/interp`
    contains neither `go/types` nor `goal/internal/typecheck`.

## T3: Verify
- `go build ./...`, `go vet ./...`, `go test ./... -count=1` all green.
- Confirm `go test ./internal/interp -run TestRun -count=1` passes.
