# Interpreter Entry over AST and sema — Business Specification

## Overview

The goscript runtime is a tree-walking interpreter that runs goal programs over
the SAME parsed AST and resolved semantic facts the Go transpiler consumes. This
story establishes the entry seam: a way to construct an interpreter from an
already-parsed, already-resolved program and run its `main` entry point. It
proves goscript is a back-end over the shared front-end (REWRITE-ARCHITECTURE.md
§3.1) — not a fork of the grammar and not a consumer of the Go backend's lowered
output.

## Functional Requirements

### FR-1: Construct from the shared front-end
The interpreter SHALL be constructible from a parsed program (an `*ast.File`)
together with its resolved semantic info (`*sema.Info`) — the same artifacts the
Go back-end receives. It SHALL NOT require the Go backend's lowered form.

### FR-2: Run the program entry point
The interpreter SHALL expose a `Run` entry that locates the top-level `func main`
(a plain function with no receiver) and executes its body. On success it SHALL
return no error.

### FR-3: Missing entry point is a named error
When the program declares no top-level `func main`, `Run` SHALL return a
descriptive, named error rather than silently succeeding.

## Acceptance Criteria

- [ ] A trivial program `package main` with an empty `func main() {}`, parsed via
      the shared parser and resolved via the shared sema pass, runs through the
      interpreter with no error.
- [ ] The construction path takes the parsed `*ast.File` plus `*sema.Info` and
      requires no Go-lowered input.
- [ ] A program with no top-level `func main` yields a descriptive error naming
      the missing entry point.

## User Interactions

No end-user surface in this story (no CLI yet — that is US-026). The interaction
is the in-process API: `interp.New(file, info)` then `interp.Run()`.

## Error Handling

- No `func main` declared → a descriptive error (e.g. naming "main" as missing).
- (Statement/expression evaluation errors are out of scope here; an empty body is
  a no-op.)

## Out of Scope

- Evaluating any statements or expressions beyond an empty body (US-005+).
- Function arguments, returns, recursion (US-007).
- The CLI `goal run --engine=interp` surface (US-026).
- Capability routing and denial (US-023/US-024).

## Open Questions

None — the seam and its single happy-path + missing-main error case are fully
determined by the acceptance criteria.
