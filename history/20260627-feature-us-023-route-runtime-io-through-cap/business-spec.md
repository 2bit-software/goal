# Route runtime IO through cap — Business Specification

## Overview

The goscript interpreter performs host effects (writing to standard output, and
in the future reading the wall clock and the environment). Today those effects
reach the operating system directly. This feature interposes the capability /
authority model between the interpreter and the host: every host effect is
routed through a capability set, so a program's power is decided by what the
host grants rather than by the language itself. By default the interpreter runs
with full authority (all capabilities granted), preserving today's behavior,
while establishing the seam that a sandbox (denial enforcement, US-024) builds
on.

## Functional Requirements

### FR-1: Effects are capability-mediated
Every host effect the interpreter performs SHALL be routed through a capability
set rather than touching the host directly. The standard-output effect is gated
by the standard-output authority; the future wall-clock and environment reads
are gated by their respective authorities.

### FR-2: Default authority is full
The default run path SHALL execute with every capability granted, so a program
that prints behaves exactly as before when the host specifies no restriction.

### FR-3: The standard-output effect targets a configurable sink
The destination of the standard-output effect SHALL be configurable by the host,
defaulting to the process's standard output. This lets a host (or a test)
capture what a program writes without depending on the process's real stdout.

## Acceptance Criteria

- [ ] All interpreter host effects are routed through a capability set; none
      writes to the host directly when capabilities mediate it.
- [ ] The default run path grants every capability.
- [ ] A program that prints under the full-grant default produces its expected
      output through the configurable sink, and that output can be captured and
      asserted.
- [ ] Existing interpreter behavior (programs that do not print) is unchanged.

## User Interactions

The runtime author constructs the interpreter, optionally supplying a capability
set and an output sink. With no options, the interpreter grants every capability
and writes to the process's standard output. A printing program's output appears
at the configured sink.

## Error Handling

Under the full-grant default, granted effects are performed normally. Refusing a
denied effect with a located, named error is intentionally NOT part of this
feature (see Out of Scope).

## Out of Scope

- Denial enforcement: a denied capability producing a located, named error and
  suppressing the effect is the next story (US-024). Here the routing seam
  exists and the default grant performs the effect.
- Capabilities with no current interpreter call site (wall clock, environment,
  filesystem, network, concurrency) are mediated by the same seam but are not
  given new effect implementations here.

## Open Questions

- None. The capability model (US-001) and the single existing effect site
  (the fmt-family stdout write) are already established in the codebase.
