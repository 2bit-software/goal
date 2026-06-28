# Audit — Completeness

Spec: business-spec.md (US-023 Route runtime IO through cap)

## Findings

- MINOR: The spec lists wall-clock and environment reads as future effects but
  there is no current interpreter call site for them. This is correctly noted in
  Out of Scope; the seam must merely be reusable for them. Not blocking.
- MINOR: "configurable sink" — the spec leaves the exact configuration mechanism
  to implementation (correct for a behavior-only spec). Acceptance is verifiable:
  print under default grant, capture, assert.
- No CRITICAL or MAJOR findings. Requirements are non-contradictory, the happy
  path (print under GrantAll → captured output) is testable, and the unchanged
  path (non-printing programs) is asserted by the existing suite.

## Assumptions

- The only effect needing conversion today is the fmt-family stdout write; the
  other capabilities have no current effect implementation (matches the codebase).
- Denial-to-error is deferred to US-024 (explicit in the PRD).
- Default sink is the process standard output; default authority is full grant.
