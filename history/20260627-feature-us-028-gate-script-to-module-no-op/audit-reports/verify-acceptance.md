# Verify: Acceptance — US-028

## Result: PASS (no CRITICAL/MAJOR findings)

Every acceptance criterion from business-spec.md is satisfied and verified by a
running test.

- AC: run sample under interpreter & capture stdout — `runUnderInterp` (interp.New
  + WithStdout + Run). VERIFIED.
- AC: transpile unchanged source, build as Go module, run, capture stdout —
  `runAsGoModule` (backend.Transpile -> temp module -> `go run .`). VERIFIED.
- AC: assert the two outputs equal and non-empty — `TestScriptToModuleNoOp`
  asserts non-empty both, equality, and that both equal "green". VERIFIED.
- AC: sample exercises a genuine goal construct — enum + value-position match.
  VERIFIED.
- AC: divergence fails loudly — each failure path `t.Fatalf`s with both outputs /
  generated Go / stderr. VERIFIED.

## Verification commands (all green)

- `go build ./...` — OK
- `go vet ./...` — OK
- `go test ./... -count=1` — OK (all packages pass)
- `go test ./internal/corpus/ -run TestScriptToModuleNoOp -v` — PASS

## Assumptions

- Observable behavior is captured stdout (trimmed). Both engines emit "green\n".
- The `go` toolchain is present in the test environment (same requirement as the
  existing corpus behavioral tier).
