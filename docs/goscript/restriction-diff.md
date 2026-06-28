# goscript restriction diff

goscript is a capability-sandboxed runtime. It does **not** restrict power by
offering a different language grammar; it restricts power by what the **host
grants**. The runtime "removes" an authority — concurrency, I/O, network — simply
by the host not granting the corresponding capability, never by changing the
language (REWRITE-ARCHITECTURE.md §4; goal-design-spec.md §4.4).

This document enumerates the capability model (`internal/cap`) and goscript's
default grant for each capability.

## The model

- A `Capability` names one host authority.
- A `CapabilitySet` holds zero or more capabilities. `Has(c)` reports membership;
  `Grant(c)` adds one.
- `GrantAll()` constructs a set holding every capability — the default authority
  goscript runs with in v1.
- `DenyAll()` constructs a set holding none — the basis a host narrows from.

## Capabilities and default grants

In v1, goscript grants **everything** by default (the run path uses `GrantAll()`).
A host narrows authority by starting from `DenyAll()` and granting only what it
chooses, or by denying specific capabilities. Enforcement at the actual effect
sites (routing stdout, time, env, etc. through the set, and refusing a denied
effect with a named error) is later runtime work, not part of this seam.

| Capability    | Authorizes                              | goscript default |
| ------------- | --------------------------------------- | ---------------- |
| `Stdout`      | Writing to standard output              | Granted          |
| `Stdin`       | Reading from standard input             | Granted          |
| `FileRead`    | Reading from the filesystem             | Granted          |
| `FileWrite`   | Writing to the filesystem               | Granted          |
| `Net`         | Opening network connections             | Granted          |
| `Concurrency` | Starting goroutines / using channels    | Granted          |
| `Time`        | Reading the wall clock                  | Granted          |
| `Env`         | Reading environment variables           | Granted          |

## Status

The grant/deny **seam** exists now (this story). Granting everything by default
keeps v1 behavior unchanged while reserving the authority model so later runtime
stories can route host effects through a `CapabilitySet` and enforce denials.
