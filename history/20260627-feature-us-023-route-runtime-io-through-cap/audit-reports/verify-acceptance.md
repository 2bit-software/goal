# Verify — Acceptance Coverage

Spec: business-spec.md (US-023)

## Full suite

`go build ./...`, `go vet ./...`, `go test ./... -count=1` — all green.
`internal/interp` and `internal/cap` pass; the US-022 dependency gate
`TestInterpHasNoGoTypesOrTypecheckDep` still passes (cap is dependency-free).

## Acceptance criteria → evidence

- "All interpreter host effects routed through a cap.CapabilitySet; none writes
  the host directly when capabilities mediate it." → `host.go` `evalHostCall`
  routes the only effect (`fmt.Println`) through `ip.emitStdout`, which checks
  `ip.caps.Has(cap.Stdout)` before writing. The `os.Stdout` literal is gone from
  host.go. Evidence: `TestEmitStdoutRoutesThroughConfiguredSink`,
  `TestPrintlnUnderGrantAllWritesToSink`.
- "Default run path grants every capability." →
  `TestNewDefaultsGrantAllCapabilities` asserts `ip.caps.Has(c)` for all eight
  capabilities. `New` sets `ip.caps = cap.GrantAll()`.
- "Print under GrantAll captured through sink." →
  `TestPrintlnUnderGrantAllWritesToSink` runs `fmt.Println("hello", 42)` with a
  `*bytes.Buffer` sink and asserts `"hello 42\n"`.
- "Existing behavior unchanged." → variadic `New(file, info, opts ...Option)`
  keeps every 2-arg call site compiling; full suite green.

## Findings
- No CRITICAL/MAJOR. All acceptance criteria have asserting tests.

## Assumptions
- The not-granted branch performs no write and returns nil (deferring the
  located, named capability-denied error to US-024, per the PRD).
