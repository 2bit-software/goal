# Enforce Capability Denial — Business Specification

## Overview

The goscript interpreter mediates every host effect through a capability set
(US-023). Today a denied capability is silently skipped: the effect simply does
not happen and the program continues as if it had. This feature makes denial
REAL — a program that attempts an effect the host has not granted is refused
loudly, by name, and at a source location, instead of having its effect quietly
dropped. This turns the capability set from advisory into an enforced sandbox.

## Functional Requirements

### FR-1: Denied capability refuses the effect

When the interpreter runs under a capability set that does NOT grant the
capability an effect requires, the effect SHALL NOT be performed and the run
SHALL surface a capability-denied error.

### FR-2: The refusal is named

The capability-denied error SHALL identify the specific capability that was
denied (e.g. Stdout), so a host or test can tell which authority was missing.

### FR-3: The refusal is located

The capability-denied error SHALL carry the source position of the effect that
was refused, consistent with the interpreter's other located refusals.

### FR-4: Nothing is written on denial

When a standard-output effect is refused for lack of the Stdout capability, no
bytes SHALL reach the output sink.

### FR-5: Granted capability is unaffected

When the required capability IS granted, the effect SHALL be performed exactly
as before (no behavioral change to the happy path).

## Acceptance Criteria

- [ ] Running a printing program under a capability set with Stdout denied
      raises a capability-denied error rather than printing.
- [ ] Under denied Stdout, the output sink receives nothing.
- [ ] The raised error names the denied capability (Stdout).
- [ ] The raised error carries a source position (located).
- [ ] Running the same printing program under a set that grants Stdout still
      prints normally and raises no error.

## User Interactions

A host constructs the interpreter with a restricted capability set (e.g.
`cap.DenyAll()` or a set without Stdout) via the existing construction option.
Running such a program returns the capability-denied error from the run.

## Error Handling

A denied effect produces a typed, located, named error surfaced as the run's
result (matchable by a host/test). It is a host-policy refusal of an effect, not
a program panic, and not a silent no-op.

## Out of Scope

- Effect sites other than standard output (time, env, file, network) — no other
  effect site is routed through the gate yet; they reuse this seam as they land.
- Changes to the capability enumeration or set semantics.
- CLI-level capability configuration (later stories).

## Open Questions

None.
