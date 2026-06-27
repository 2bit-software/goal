# Audit — AI-Consumer Readiness

## Findings

No CRITICAL findings.
No MAJOR findings.

### Assessment

- Terms are defined: capability set, denied capability, effect, sink all map to
  existing `internal/cap` / `internal/interp` constructs.
- Acceptance criteria are directly assertable: deny Stdout -> expect error +
  empty sink; grant Stdout -> expect output + nil error.
- The error contract (named + located) is specific enough to assert on (the
  capability name appears in the message / is carried on a typed error; a source
  position string is present).
- No clarifying questions required to implement.

## Assumptions

- The error is named by including the `cap.Capability.String()` value (e.g.
  "Stdout") in its message and/or as a typed field.
- The test constructs the restricted interpreter via the existing
  `WithCapabilities` option with `cap.DenyAll()` (or a set lacking Stdout) and a
  captured `WithStdout` sink to assert emptiness.
