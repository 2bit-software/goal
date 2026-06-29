# Completeness Audit — US-004

## Findings

### MINOR — emit-output cleanliness not specified
The spec does not say where bootstrap artifacts live or how they avoid breaking
`task check`/`task build`. FR-4 requires the gates stay green; the technical
research resolves this (emit under a `_`-prefixed dir the Go toolchain ignores
under `./...`). Not a spec gap, just a reminder for implementation.

### MINOR — re-run idempotence
The spec does not state the tasks must be re-runnable. Implementation should
clear prior emit dirs at the start of each run so stale output cannot mask a
real difference. Behavior-preserving; no spec change needed.

## Assumptions

- The bootstrap drives the existing `goal build --emit=<dir> <path>` CLI
  contract unchanged; the skeleton replicates that emit layout.
- The skeleton imports the existing Go `internal/*` packages (ported packages
  arrive in US-005+), so goal-c-1 and goal-c-2 are functionally identical and
  the fixpoint is trivially byte-identical at this stage.
- Bootstrap artifacts and the goal-c binaries are build outputs, not committed.
