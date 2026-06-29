# Per-package transpile-and-build smoke gate — Business Specification

## Overview

The self-host effort needs continuous proof that the goal front-end transpiles
the compiler's own packages into Go that actually compiles. The static checker is
silent on a class of transpile defects (the iota miscompile fixed in US-001 was
found only because the generated Go failed to build), so `go build` over the
generated output is the real proof. This feature adds a repeatable gate that
transpiles each in-scope compiler package through the goal front-end and compiles
the result, failing loudly if any package produces non-compiling Go.

## Functional Requirements

### FR-1: Transpile-and-build each in-scope package
The gate SHALL, for each in-scope compiler package, transpile that package's
source through the goal front-end and compile the generated Go with `go build`.

### FR-2: Coverage
The gate SHALL cover at least: token, lexer, ast, parser, sema, project, pipeline,
backend.

### FR-3: Green today, red on regression
The gate SHALL pass against the current source tree (after US-001) and SHALL fail
if any covered package transpiles to Go that does not compile.

### FR-4: Runs under the standard verify gates
The gate SHALL run as part of the project's existing verification commands
(`task check`) so it executes on every routine check, with no extra third-party
dependencies.

## Acceptance Criteria

- [ ] A test transpiles each in-scope package's source through the goal front-end
      and runs `go build` on the generated Go.
- [ ] The gate covers token, lexer, ast, parser, sema, project, pipeline, backend.
- [ ] The gate is green against the current tree.
- [ ] The gate fails (red) when a covered package transpiles to non-compiling Go,
      demonstrated by an explicit negative case.
- [ ] `task check` and `task build` are green.

## User Interactions

Developers run the gate via `task check` (or `go test`) — no new CLI surface. A
failure names the offending package and surfaces the `go build` output.

## Error Handling

- A package that fails to transpile (front-end error) fails the gate with the
  front-end's located error.
- A package that transpiles but does not compile fails the gate with the captured
  `go build` output.

## Out of Scope

- Running each package's existing unit tests against the transpiled output (the
  story says "where practical"; that is the explicit job of the later per-package
  port stories US-005+). This gate proves *compilation*, the silent-defect detector.
- Porting any package to goal source (US-005+).
- Rewriting `ast/dump.go`'s reflect usage (US-007).

## Open Questions

- None blocking. Note for downstream: the PRD premise "the compiler source is
  already valid goal" is false where the source uses goal reserved words
  (`enum`/`match`/`assert`) as plain identifiers; making the gate green entails a
  small, behavior-preserving rename of such identifiers in the covered packages.
