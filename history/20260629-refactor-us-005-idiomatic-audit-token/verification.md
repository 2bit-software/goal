# Verification — US-005 Idiomatic audit: token

All acceptance criteria verified green.

| AC | Check | Result |
|----|-------|--------|
| AC-1: enum-or-record | DECISIONS.md entry recording the deliberate decision to keep `Kind` as an iota const block (sealed-interface enum encoding does not fit an array-indexed, range-marker integer Kind) | PASS — entry appended under "self-host idiomatic audit — US-005 (token)" |
| AC-2: no goal-fix sites | `goal fix selfhost/token/token.goal` | PASS — no diff, no stderr report |
| AC-3: token tests | `task check` (incl. internal/selfhost port gate transpiling selfhost/token and running internal/token tests against it; internal/token) | PASS — `ok goal/internal/selfhost`, `ok goal/internal/token` |
| AC-4: fixpoint | `task fixpoint` | PASS — `FIXPOINT OK` (goal-c-1/goal-c-2 byte-identical, selfhost/token/token.go included) |
| AC-5: build/check | `task build`, `task check` | PASS — both green |

## Notes
- No `.goal` source change: the audit's correct outcome is the recorded decision
  (AC-1's escape hatch). token.goal already satisfied AC-2/AC-3/AC-4.
- Behavior preserved: the verbatim US-003 self-host oracle stays green
  (fixpoint byte-identical; reused internal/token tests pass against the
  transpiled package).
