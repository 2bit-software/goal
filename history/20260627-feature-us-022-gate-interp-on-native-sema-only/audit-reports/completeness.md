# Audit: Completeness — US-022

## Findings

No CRITICAL or MAJOR findings.

- MINOR: The spec surfaces the first Error-severity diagnostic only. This is
  explicitly noted as Out of Scope (rendering all diagnostics), so it is not a
  gap — a single located refusal is sufficient to prove the gate.

All four functional requirements are testable; acceptance criteria cover the
happy path (clean program runs), the error path (non-exhaustive match
refused), the warning case (does not block), and the dependency envelope.

## Assumptions

- The first Error diagnostic in source order is the one reported. sema.Check
  aggregates checks in a fixed order and collectMatches is source-ordered, so
  this is deterministic.
- "located" is satisfied by token.Pos.String() => "line:col".
