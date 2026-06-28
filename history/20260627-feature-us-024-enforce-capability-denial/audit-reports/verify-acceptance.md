# Verification — Acceptance Coverage

Full suite (prd.json verifyCommands) green:
- `go build ./...` — OK
- `go vet ./...` — OK
- `go test ./... -count=1` — all packages ok (interp, cap, corpus, backend, ...).

## Criterion -> evidence

| Acceptance criterion | Test | Asserts |
|----------------------|------|---------|
| Denied capability refuses the effect with a located, named error | `TestPrintlnUnderDeniedStdoutIsRefused` (cap_deny_test.go) | Run under DenyAll returns an error that errors.As into CapabilityError |
| Error names the denied capability (Stdout) | same | `capErr.Cap == cap.Stdout` and message contains "Stdout" |
| Error is located | same | message contains `capErr.Pos.String()` (non-empty) |
| Nothing is written under denial | same | `buf.Len() == 0` |
| Write func is not invoked under denial (gate-level) | `TestEmitStdoutDeniedReturnsLocatedNamedErrorAndSkipsWrite` | `wrote == false`; error located at the supplied 7:2 |
| Granted capability still performs the effect (FR-5) | `TestPrintlnUnderGrantedStdoutStillPrints` | Run under GrantAll returns nil; buf == "hello 42\n" |

All acceptance criteria have a corresponding asserting test. No criterion is
left to manual inspection.
