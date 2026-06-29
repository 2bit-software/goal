# Plan Coverage Audit — US-002

- FR-1 (transpile+build each package) -> `BuildTranspiled` + test. Covered.
- FR-2 (coverage list) -> `InScope` with all 8 packages. Covered.
- FR-3 (green now / red on regression) -> positive + negative tests. Covered.
- FR-4 (runs under verify gates) -> Go test in `internal/selfhost`, runs under
  `task check`. Covered.
- The `enum`-identifier renames trace to FR-3 ("green now") — without them sema and
  backend do not transpile. Not scope creep; prerequisite, empirically confirmed.

No CRITICAL or MAJOR findings.

## Assumptions
- A new `internal/selfhost` package is an acceptable home (vs. piggybacking on an
  existing test package). Chosen so later port stories reuse the harness.
