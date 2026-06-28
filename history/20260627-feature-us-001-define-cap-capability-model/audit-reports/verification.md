# Verification — US-001 Define cap capability model

## verifyCommands (prd.json) — all green
- `go build ./...` — clean
- `go vet ./...` — clean
- `go test ./... -count=1` — all packages ok, including new `goal/internal/cap`

## Acceptance criteria
- [x] internal/cap defines a Capability enumeration covering Stdout, Stdin,
      FileRead, FileWrite, Net, Concurrency, Time, Env — `internal/cap/cap.go`.
- [x] CapabilitySet with Has/Grant and GrantAll()/DenyAll() constructors —
      `internal/cap/cap.go`.
- [x] Unit test asserts GrantAll().Has(c) true and DenyAll().Has(c) false for
      every defined Capability — `internal/cap/cap_test.go`
      (TestGrantAllHasEvery / TestDenyAllHasNone iterate allCapabilities()).
- [x] docs/goscript/restriction-diff.md enumerates each capability and whether
      goscript grants it by default — present, table lists all eight as Granted.

## Notes
- Stdlib `testing` only (no testify) per project zero-dependency constraint.
- v1 grants all by default (GrantAll); deny seam exists for later enforcement
  stories, as the spec scopes.

## Result: PASS
