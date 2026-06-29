# US-004 Bootstrap and fixpoint harness — Business Specification

## Overview

The self-host effort needs the classic 3-stage compiler bootstrap and the
byte-identical fixpoint trust proof available as repeatable, one-command
targets. This lets every later compiler-package port be validated against the
trust proof from the start. This story stands up the harness against a thin
skeleton of the goal-written compiler; later stories grow that skeleton.

## Functional Requirements

### FR-1: Goal-written compiler skeleton
The repository SHALL contain a goal-source `package main` program that behaves
as a `goal build --emit` equivalent: given an output directory and a source
path, it transpiles every goal package found under the path and writes the
generated Go under the output directory, mirroring the existing toolchain's
emit layout.

### FR-2: Bootstrap target
There SHALL be a `bootstrap` task that runs the 3-stage sequence: build the
trusted Go-built compiler (stage 0), use it to transpile and build the
goal-written compiler (goal-c-1), then use goal-c-1 to transpile and build the
goal-written compiler again (goal-c-2).

### FR-3: Fixpoint target
There SHALL be a `fixpoint` task that has goal-c-1 and goal-c-2 each emit Go for
the compiler's own source and compares the two emissions with `diff -r`. The
task SHALL exit 0 when, and only when, the two emissions are byte-identical.

### FR-4: Project gates stay green
Running the harness SHALL NOT break the project-wide `task check` and
`task build` gates.

## Acceptance Criteria

- [ ] A goal-source `package main` program exists that transpiles goal packages
      under a path and writes the generated Go under a chosen output directory.
- [ ] A `bootstrap` task runs stage-0 -> goal-c-1 -> goal-c-2 end to end and
      exits 0.
- [ ] A `fixpoint` task runs the bootstrap sequence, diffs the goal-c-1 and
      goal-c-2 emissions for the compiler's own source, and exits 0 when they are
      byte-identical.
- [ ] If the two emissions ever differ, the `fixpoint` task exits non-zero.
- [ ] `task check` and `task build` remain green with the harness present.

## User Interactions

- `task bootstrap` — runs the 3-stage bootstrap.
- `task fixpoint` — runs the bootstrap then proves byte-identical fixpoint.

## Error Handling

- A transpile failure, a Go build failure, or a non-identical emission SHALL
  surface as a non-zero exit from the relevant task, with the failing stage's
  output shown.

## Out of Scope

- Porting any compiler package (token, lexer, ast, ...) to goal — that is
  US-005 onward. The skeleton imports the existing Go `internal/*` packages.
- Achieving a meaningful differential fixpoint over ported packages — for the
  skeleton the two emissions are trivially identical.
- Self-hosting `goal check`/typecheck (US-013).

## Open Questions

- None. The CLI contract and emit layout already exist in the toolchain and are
  reused as-is.
