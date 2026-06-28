# US-022 Gate interp on native sema only — Business Specification

## Overview

The goscript tree-walking interpreter must refuse to run any program that
violates a static guarantee, computing that refusal solely from its own native
semantic checks (internal/sema, which reads the parsed AST), never from the
go/types depth checker. This lets goscript run in a host with no Go toolchain:
types are checked statically and erased at runtime, and a violated guarantee
fails loudly before evaluation rather than being silently mis-run.

## Functional Requirements

### FR-1: Native sema gate before evaluation
The interpreter's run path SHALL run the native sema checks over its parsed
AST + resolved Info, and if any Error-severity diagnostic is present, SHALL
refuse the program BEFORE evaluating `func main`.

### FR-2: Located, named refusal
The refusal SHALL be a located, named error: it SHALL carry the source
position (line:col) and the diagnostic's stable code and message, so a non-
exhaustive `match` is reported with its location, not silently run.

### FR-3: Warnings do not block
A Warning-severity diagnostic (a located deferral, e.g. an unresolved-enum
match) SHALL NOT block execution — only Error-severity guarantees refuse.

### FR-4: Native-only dependency envelope
internal/interp SHALL NOT depend on internal/typecheck or go/types.

## Acceptance Criteria

- [ ] Running a non-exhaustive-match program through the interpreter returns a
      located diagnostic error (line:col present) and does NOT evaluate main.
- [ ] Running a sema-clean program is unaffected (no false refusal).
- [ ] A Warning-only program is not refused.
- [ ] A dependency test (`go list -deps ./internal/interp` or source scan)
      asserts internal/interp does not depend on go/types or
      goal/internal/typecheck.

## User Interactions

No new CLI surface. The gate is internal to the interpreter's run path;
callers (tests today, the CLI in US-026) observe a returned error instead of
a successful run for a guarantee-violating program.

## Error Handling

A guarantee violation surfaces as a single returned error naming the first
Error-severity diagnostic (location + code + message). A program with only
Warnings or no diagnostics runs normally.

## Out of Scope

- Adding new sema checks (the existing aggregate sema.Check is reused as-is).
- Capability/IO routing (US-023/024).
- Rendering ALL diagnostics; surfacing the first Error is sufficient to refuse.

## Open Questions

None.
