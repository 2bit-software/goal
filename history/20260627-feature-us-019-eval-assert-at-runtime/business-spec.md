# Eval assert at runtime — Business Specification

## Overview

The goscript tree-walking interpreter must honor goal's `assert` statement.
`assert` exists so a runtime invariant the static type system cannot capture
still fails loudly — never silently — when the program runs under
interpretation. A true condition lets evaluation continue; a false condition
stops the program with a located panic carrying the assertion message.

## Functional Requirements

### FR-1: True assertion is a no-op
When an `assert` condition evaluates to a true boolean, the statement has no
observable effect and evaluation continues with the following statements.

### FR-2: False assertion panics, located, with a message
When an `assert` condition evaluates to a false boolean, the interpreter raises
a loud runtime panic that propagates to the host (recovered at no intermediate
boundary). The panic message identifies the failure as an assertion failure,
carries the source location of the assert, and includes the asserted condition.

### FR-3: printf-message form is formatted into the panic
For the message form `assert <cond>, "<format>", <args>...`, when the condition
is false the formatted message (the format string applied to the args) is
appended to the panic message. Commas nested inside the condition do not split
the message — only the top-level comma after the condition does (already handled
by the parser).

### FR-4: Non-boolean condition is a descriptive refusal
An `assert` whose condition does not evaluate to a boolean is a descriptive,
named error rather than a silent pass — consistent with the interpreter's other
condition sites (e.g. `if`).

## Acceptance Criteria

- [ ] An `assert` with a true condition completes normally and the following
      statements run.
- [ ] An `assert` with a false condition stops evaluation with a panic whose
      message marks an assertion failure and includes the source location.
- [ ] A false `assert` in the printf-message form includes the formatted message
      text in the panic.
- [ ] A non-boolean `assert` condition produces a descriptive error, not a
      silent no-op.
- [ ] A unit test over a 10-assert-shaped program asserts the panic on a false
      assertion and normal completion on a true one.

## User Interactions

No new surface. `assert` is existing goal source syntax; this story makes the
interpreter execute it. Authors run `.goal` programs containing `assert` through
the interpreter and observe loud failure on a violated invariant.

## Error Handling

- False assertion → loud panic propagated to the host (the existing panic
  channel), message `assertion failed: <condition>[: <formatted message>]` plus
  source location.
- Non-boolean condition → descriptive interpreter error naming the offending
  kind.

## Out of Scope

- A build mode that disables assertions (asserts are always active under interp).
- Capability mediation of the failure output (covered by US-023/US-024).

## Open Questions

None.
