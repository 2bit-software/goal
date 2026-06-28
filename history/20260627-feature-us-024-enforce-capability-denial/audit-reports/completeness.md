# Audit — Completeness

## Findings

No CRITICAL findings.
No MAJOR findings.

### MINOR

- The spec covers only the Stdout effect site because that is the only effect
  routed through the gate today. This is correctly captured under Out of Scope;
  noting it here so the narrow happy-path/denial coverage is intentional, not a
  gap.

## Coverage check

- Happy path (granted): FR-5 + last acceptance criterion. ✓
- Denial path (refused, named, located, nothing written): FR-1..FR-4 + first
  four acceptance criteria. ✓
- Requirements use SHALL/SHALL NOT; no "might/usually/should" ambiguity. ✓
- No contradictions; no dependency on open questions (none remain). ✓

## Assumptions

- A denial is surfaced as a TYPED RETURNED error (matchable via errors.As), not
  a panic — chosen because capability denial is a host-policy refusal of an
  effect, not a program fault. Validated against the interpreter's existing
  located-refusal style (gate(), unresolved-host-symbol).
- "Located" means a source position string carried on the error; the only
  routed effect site (fmt.Println) has the selector position available.
