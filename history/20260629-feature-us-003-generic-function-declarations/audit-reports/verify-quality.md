# Verify — Quality

- Change reuses existing parseTypeParams/atTypeParams and fieldList helpers — no
  duplicated logic.
- New field nil-guarded everywhere; non-generic path byte-identical.
- Test asserts emitted text for both unconstrained and constrained params and
  validates the output is valid Go via go/format.

No CRITICAL/MAJOR findings.

## Assumptions
- go/format parsing (full Go generics grammar) plus the manual goalc+go build
  smoke is sufficient evidence of "valid Go" without adding a permanent
  build-harness test.
