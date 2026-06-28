# Wire interpreter into the CLI — Business Specification

## Overview

Goal users today reach the goscript interpreter only through tests; the `goal`
command can only transpile to Go and shell out to the Go toolchain. This feature
exposes the tree-walking interpreter on the command line so a user can run a
`.goal` program directly under interpretation, without a Go toolchain step.

## Functional Requirements

### FR-1: Interpreter engine selection
`goal run --engine=interp <file.goal>` SHALL execute the named program under the
goscript interpreter: parse it, validate it through the native semantic checks,
and run its `func main`.

### FR-2: Success exit
On successful completion of `func main`, the interpreter run SHALL succeed (the
command exits 0).

### FR-3: Program output
Standard-output produced by the program (e.g. `fmt.Println`) SHALL appear on the
command's standard output.

### FR-4: Default behavior unchanged
Without `--engine=interp` (no flag, or `--engine=ast`), `goal run` SHALL retain
its existing transpile-and-`go run` behavior. The interpreter is strictly opt-in.

### FR-5: Loud failure
A program that fails to parse, lacks a `func main`, violates a static guarantee
the native sema gate enforces (e.g. a non-exhaustive match), or fails at runtime
SHALL cause the command to exit non-zero with a descriptive message — never a
silent success.

## Acceptance Criteria

- [ ] `goal run --engine=interp <file.goal>` on a program that prints via
      `fmt.Println` exits 0 and prints the expected text to stdout.
- [ ] An unknown `--engine` value is rejected with a descriptive error.
- [ ] `goal run` without `--engine=interp` behaves exactly as before.
- [ ] A program with no `func main` run under `--engine=interp` exits non-zero
      with a descriptive error.

## User Interactions

CLI: `goal run --engine=interp <path-to-.goal-file>`.

## Error Handling

Parse, gate (static-guarantee), no-main, and runtime failures are returned as a
non-nil command error, which the binary reports to stderr and exits 1.

## Out of Scope

- Running a multi-file package or directory under the interpreter (the interp is
  single-`*ast.File`; the path is one `.goal` file).
- Adding `--engine=interp` to `goal build`/`goal check`.
- Capability narrowing flags (the CLI run uses full authority / GrantAll).

## Open Questions

- None.
