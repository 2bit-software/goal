# Implementation Tasks

## Task 1: Confirm the machine check (no auto-convertible sites)
**Status**: completed
**Files**: (read-only) selfhost/typecheck/*.goal
**Depends on**: (none)
**Spec coverage**: FR-2, AC-2
**Verify**: `for f in selfhost/typecheck/*.goal; do ./bin/goal fix "$f"; done` produces no
source diff (stdout equals input; only `skipped`/`suggestion` advisories on stderr).

### Instructions
Run `goal fix` over each `.goal` file in the package and confirm the emitted output is
byte-identical to the input. The only stderr reports must be the deliberate non-conversions:
a `skipped: [result-sig]` on `Load` and advisory `suggestion` call-site hints on `Load` and
`Check`. None is an auto-conversion, so AC-2 holds with no source change.

## Task 2: Record the refusals in DECISIONS.md
**Status**: completed
**Files**: DECISIONS.md
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-3
**Verify**: `grep -n "US-012" DECISIONS.md` shows the new section.

### Instructions
Append a `## self-host idiomatic audit — US-012 (typecheck)` section mirroring the US-010
format. Document three refusals with reasons:
- `Load` stays `(*Package, error)` — exported, oracle-pinned, in-tree caller; wrapping
  propagation is not a `?` site; callees are Go-tuples not Result.
- `GoTypesChecker.Check` stays `([]Diagnostic, error)` — interface method; pure propagation
  but the host cannot become Result without changing the `TypeChecker` contract.
- `litClass` stays `type litClass int` + iota — no switch over it; consumed via `==` and a
  numeric `return 0`; enum lowers to a sealed interface with no integer identity.
Close with a Verification subsection listing the green commands.

## Task 3: Verify project gates
**Status**: completed
**Files**: (none — verification only)
**Depends on**: Task 2
**Spec coverage**: FR-4, AC-3, AC-4
**Verify**: `task check` (incl. the selfhost port gate + internal/typecheck depth tests),
`task build`, `task fixpoint` all green.

### Instructions
Run the prd.json verifyCommands in order. The port gate transpiles `selfhost/typecheck` and
runs the copied `internal/typecheck` depth tests against it; `task fixpoint` proves
goal-c-1/goal-c-2 emit byte-identical Go (the package is unchanged, so this must stay green).
