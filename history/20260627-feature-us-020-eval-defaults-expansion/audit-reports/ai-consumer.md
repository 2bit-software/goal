# Audit: AI-Consumer Readiness — US-020

## Findings

No CRITICAL findings.
No MAJOR findings.

The spec is directly implementable without guessing:

- All terms are defined. `...defaults`, "safe zero value", and the per-type zero
  mapping are concrete (FR-3 gives the full table).
- Data formats are specified: the per-type zero is enumerated; struct fields and
  their declared types are read from the resolved front-end facts.
- State transitions are explicit: construct struct -> for each declared field not
  explicitly set, fill its zero -> return the struct.
- Acceptance criteria are specific enough to write assertions from (each names a
  concrete field type and its expected zero, plus a refusal case).

### MINOR-1: source of field types not user-facing
The spec intentionally omits HOW the interpreter learns a struct's declared
fields (it reads resolved sema facts). This is correct for a behavior spec; the
technical-requirements doc records the mechanism.

## Assumptions

- "Safe zero" is defined exactly as the existing front-end defines it (sema
  CheckFields + backend zeroLit); the interpreter mirrors that definition rather
  than inventing a new one.
- Single-file resolution: the struct being defaulted is declared in (or resolved
  for) the same program under interpretation, consistent with the rest of the
  interp stories.
