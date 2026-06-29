# US-010 Port project and pipeline packages to goal — Business Specification

## Overview

The self-host effort is porting the goal compiler's own packages from Go to
goal-native source, leaf-to-root through the dependency DAG. With token, lexer,
ast, parser, and sema already ported (US-005..US-009), this story ports the two
small remaining front-end support packages: `internal/project` (package
discovery — locating and grouping `.goal` files by directory and package name)
and `internal/pipeline` (the engine-independent output types plus the `//line`
source-position mapping). Both become goal-native source under `selfhost/`.

## Functional Requirements

### FR-1: Goal-native project package
`selfhost/project` SHALL hold the project package as goal source equivalent to
`internal/project`.

### FR-2: Goal-native pipeline package
`selfhost/pipeline` SHALL hold the pipeline package (output types + source map)
as goal source equivalent to `internal/pipeline`.

### FR-3: Transpiles to compiling Go
Both ported packages SHALL transpile through the goal front-end (the US-002
smoke gate) and the generated Go SHALL compile, with io/fs, os, path/filepath
(and fmt, sort, strings) passing through as foreign imports.

### FR-4: Behavioral equivalence
The existing self-contained project and pipeline test suites SHALL pass against
the transpiled output.

## Acceptance Criteria

- [ ] `selfhost/project` and `selfhost/pipeline` hold those packages as goal source.
- [ ] Both transpile and the generated Go compiles (io/fs, os, path/filepath pass through).
- [ ] The existing project and pipeline tests pass against the transpiled packages.
- [ ] `task check` and `task build` are green.
- [ ] `task fixpoint` stays byte-identical with the new selfhost packages included.

## User Interactions

No end-user interface. The deliverable is verified through the
`internal/selfhost` port_test gates (BuildTranspiled + BuildAndTest) run via
`go test ./internal/selfhost`.

## Error Handling

If a ported package transpiles to non-compiling Go, the BuildTranspiled gate
fails the test. If behavior diverges, the BuildAndTest gate fails when the
existing tests run against the transpiled output.

## Out of Scope

- Wiring selfhost/project or selfhost/pipeline into a goal-written main (US-012).
- The corpus-dependent pipeline_test.go suite (needs backend + corpus + the
  repo-relative manifest fixture, which the throwaway temp module cannot satisfy).
- Any behavior change to the Go-side internal/project or internal/pipeline.

## Open Questions

None. The port pattern and test-suite selection are established by US-005..US-009.
