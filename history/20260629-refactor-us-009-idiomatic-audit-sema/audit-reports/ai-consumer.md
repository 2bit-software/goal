# Audit: AI-Consumer Readiness — US-009 sema

## Findings

### MINOR — terms are defined elsewhere
Terms like ModeResult, comma-ok, oracle-pinned, and `?` propagation are domain
jargon, but each is defined in progress.txt's Codebase Patterns and DECISIONS.md
and used consistently. An implementer with repo context can act without guessing.

The acceptance criteria are all machine-checkable:
- `goal fix selfhost/sema/*.goal` → no source diff.
- `task check` / `task build` / `task fixpoint` → green, fixpoint byte-identical.
This makes the spec directly testable.

No CRITICAL or MAJOR findings.

## Assumptions

- The conversion of `Analyze` uses goal's Result construction (`return
  Result.Ok(...)`) and postfix `?` exactly as the frontend already lowers them
  (verified present in backend emit lowering); no new language feature is needed.
- DECISIONS.md is the canonical home for recorded refusals (consistent with
  US-005..US-008 audit sections).
