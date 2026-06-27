# Verification — Quality

## Checks

- Error handling: the denied branch returns a typed CapabilityError BEFORE
  invoking the write func — no partial write, no silent nil. Matches the spec's
  "fail-not-silent" intent.
- Tests assert what they claim: the denial test checks both the error (typed +
  named + located) AND the empty sink; the gate-level test independently proves
  the write func is skipped (not merely that an error was returned); the
  regression test pins the unchanged granted path.
- No contradiction with spec: granted path behavior is byte-identical to US-023
  (TestPrintlnUnderGrantAllWritesToSink / TestEmitStdoutRoutesThroughConfiguredSink
  still pass).
- US-022 dependency envelope intact: only cap + token (both dependency-free)
  added to the gate; `go vet ./internal/interp/...` clean and the full suite
  green, including TestInterpHasNoGoTypesOrTypecheckDep.

## Edge cases

- Position unavailable at a direct call: token.Pos{} renders "0:0" — acceptable
  for the gate-level test; the real fmt.Println site always has sel.Pos().
- Only Stdout is routed today; the typed error generalizes to other capabilities
  by construction (Cap field), so future effect sites need no new error type.

## Findings

No CRITICAL, MAJOR, or MINOR findings. Implementation matches the spec.
