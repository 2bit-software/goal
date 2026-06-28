# Research — US-001 Define cap capability model

## Authoritative source: REWRITE-ARCHITECTURE.md

- §4 (line 266): "Capabilities are authority, not language difference (spec §4.4):
  goscript 'removes' concurrency/I/O by the host not granting the capability, not by
  a different grammar. **Reserve the `cap` seam now.**"
- §3 table (line 196): `cap` = "capability/effect model (shape now, surfaced later)".
- §4 (line 269): "The restriction diff is still an open spec item ... 'enumerate
  exactly what goscript removes.' ... it must be written before backend/interp."
- §7 Phase 5 (line 367): "Write the goscript restriction diff; add cap + backend/interp."

Conclusion: this story builds ONLY the seam + the diff doc. v1 grants everything
by default (GrantAll is the default the later Run path will use, per US-023). The
diff doc enumerates each capability and records that goscript v1 grants it by
default — the deny path exists but is opt-in by the host.

## Standard library only

The codebase is zero-dependency (progress.txt Codebase Patterns; MEMORY.md
"No testify"). A capability set is naturally a bitset over a small enum, which is
pure stdlib. No external research warranted.

## Design decision: representation

- `Capability` is an `int`-backed enum (iota), one constant per capability.
- `CapabilitySet` wraps a `uint64` bitmask (bit `1 << cap`). Cheap Has/Grant,
  trivially copyable by value, and GrantAll/DenyAll are constant-time.
- `allCapabilities()` returns the ordered slice of every defined Capability so the
  unit test can iterate exhaustively and GrantAll can set every bit. Adding a new
  capability means adding to the enum AND the slice — guard with a test that the
  slice length matches the iota count is optional but the all-cap test covers it.
- `Grant(c)` mutates the receiver (pointer) and is the additive primitive;
  GrantAll/DenyAll are package-level constructors returning a CapabilitySet value.

## Confidence: High

The acceptance criteria are fully prescriptive; the architecture doc fixes the
semantics (authority model, grant-all default). No open questions.
