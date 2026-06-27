# Cap Capability Model — Business Specification

## Overview

goscript is a capability-sandboxed runtime: it restricts power by what the host
grants, not by offering a different language grammar. This feature introduces the
authority seam — a named set of capabilities and a capability set that can hold,
grant, and deny them — together with documentation enumerating each capability and
goscript's default grant for it. v1 grants everything by default; the value of this
story is that the grant/deny seam exists now so later runtime work can route host
effects through it.

## Functional Requirements

### FR-1: Capability enumeration
The system SHALL define a named capability for each host authority the runtime can
be granted, covering at least: standard output, standard input, file read, file
write, network, concurrency, time, and environment access.

### FR-2: Capability set membership
The system SHALL provide a capability set that reports whether it currently holds a
given capability.

### FR-3: Granting
The system SHALL allow an additional capability to be granted into a capability set.

### FR-4: All-granting and all-denying constructors
The system SHALL provide one constructor producing a set that holds every defined
capability, and one constructor producing a set that holds none.

### FR-5: Exhaustive grant/deny correctness
For every defined capability, the all-granting set SHALL report it as held and the
all-denying set SHALL report it as not held.

### FR-6: Restriction-diff documentation
The system SHALL include documentation enumerating each capability and stating
whether goscript grants it by default.

## Acceptance Criteria

- [ ] A capability exists for each of: Stdout, Stdin, FileRead, FileWrite, Net,
      Concurrency, Time, Env.
- [ ] A capability set can answer "do you hold capability X?" for any capability.
- [ ] A capability can be granted into a set, after which the set holds it.
- [ ] An all-granting constructor and an all-denying constructor both exist.
- [ ] For every defined capability, all-granting holds it and all-denying does not
      (verifiable by iterating every capability).
- [ ] Documentation enumerates each capability and its default goscript grant.

## User Interactions

This is an internal runtime seam with no direct end-user surface. Its consumers are
later runtime stories (the interpreter routes host effects through a capability set;
the default run path uses the all-granting set). The documentation artifact is the
human-facing surface.

## Error Handling

Membership queries are total: an unheld capability simply reports not-held. This
story does not itself raise capability-denied errors — that enforcement is a later
story; here the seam only needs to answer membership correctly.

## Out of Scope

- Routing any actual host effect (stdout, time, env) through the capability set.
- Raising capability-denied errors at effect sites.
- Per-capability configuration beyond hold/not-hold (e.g. path allowlists).
- Wiring the capability set into the interpreter or CLI.

## Open Questions

None. The acceptance criteria are fully prescriptive and the authority semantics
are fixed by REWRITE-ARCHITECTURE.md §4 (capabilities are authority, v1 grants all).
