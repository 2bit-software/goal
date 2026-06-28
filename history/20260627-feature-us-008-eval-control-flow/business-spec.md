# US-008 Eval control flow — Business Specification

## Overview

The goscript tree-walking interpreter must evaluate the core Go control-flow
statements so branching and looping programs run correctly under interpretation.
if/else evaluation already exists; this story adds for loops, switch, nested
block scoping, and break/continue, completing ordinary control flow ahead of the
composite-type and goal-specific stories.

## Functional Requirements

### FR-1: for loops
The interpreter SHALL evaluate three-clause for loops (`for init; cond; post {}`),
condition-only loops (`for cond {}`), and infinite loops (`for {}`). The loop
condition, when present, SHALL be a boolean; a non-bool condition is a refusal.
The init statement runs once before the loop; the post statement runs after each
iteration.

### FR-2: switch
The interpreter SHALL evaluate expression switches: tagged switches dispatch to
the first case whose expression equals the tag; tagless switches (`switch {}`)
dispatch to the first case whose expression is true. A default clause runs when
no case matches. Selected clauses do NOT fall through.

### FR-3: nested block scoping
The interpreter SHALL run a bare nested block and each loop/switch/if body in its
own lexical scope, so a variable declared inside does not leak to the enclosing
scope and inner declarations shadow outer ones.

### FR-4: break / continue
The interpreter SHALL evaluate break (exits the nearest enclosing for loop or
switch) and continue (advances the nearest enclosing for loop to its post clause
and next iteration). A break inside a switch exits only the switch; a continue
inside a switch propagates to the enclosing loop.

## Acceptance Criteria

- [ ] A summation for loop (`for i := 0; i < n; i++ { sum += i }`) yields the
      correct total.
- [ ] A condition-only loop and an infinite loop terminated by break both run.
- [ ] A tagged switch dispatches to the correct case and to default when no case
      matches.
- [ ] A tagless switch dispatches on the first true case.
- [ ] continue skips the remainder of a loop body; break ends the loop early.
- [ ] A variable declared in a nested block does not exist in the enclosing scope.
- [ ] An if/else chain selects the correct branch (regression — already present).

## User Interactions

No direct user interface. Behavior is observed by constructing a goal program,
running it through internal/parser + internal/sema, and evaluating with the
interpreter (or by evaluating statements directly against a scope in unit tests).

## Error Handling

- A non-bool for/if condition is a descriptive, named error.
- break/continue outside any loop/switch surface as a descriptive error rather
  than a silent no-op or a panic.
- Unsupported control forms (goto, fallthrough, labelled break) remain
  descriptive refusals.

## Out of Scope

- for-range over slices/maps (RangeStmt) — US-009.
- goto, fallthrough, and labelled break/continue.
- type switches.

## Open Questions

None — the seam (error-sentinel control signals) and Go semantics are settled.
