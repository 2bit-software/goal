# AI-Consumer Readiness Audit — US-008

## Findings

An AI agent can implement this without guessing:

- Data shapes are defined: a doctest Case carries Input (shared with its
  transpile twin) and transpiles to `Output.Go` (package) + `Output.Test`
  (sidecar). These types already exist in `internal/corpus`.
- The temp-module recipe is fully precedented in `RunCompile`
  (behavior_runner.go): minimal go.mod, write files, exec `go <verb> ./...`.
- Acceptance criteria are specific enough to assert against (`go test` passes
  per case; loud zero-case guard).

No CRITICAL or MAJOR findings.

## Assumptions

- The runner lives in `internal/corpus` alongside the other runners and reuses
  the `Transpiler` seam (no new interface).
- The whole-corpus test is `-short`-skipped like the compile tier, since it
  spawns the go toolchain per case.
