# Completeness Audit — US-005

## Findings

- **MINOR**: The spec does not state where the verification test lives. Resolved
  by convention: the US-002 harness lives in internal/selfhost; the new ported
  test belongs there too. Not blocking.
- **MINOR**: "existing token tests pass against the transpiled package" — the
  internal/token tests are same-package (white-box, reference unexported
  beg/end markers). Verification must compile the transpiled Go in the SAME
  package directory as the copied test file. Noted for implementation.

No CRITICAL or MAJOR findings. The spec is testable; acceptance criteria map
directly to test assertions.

## Assumptions

- The token source is copied verbatim from internal/token (Go superset is valid
  goal); no semantic edits.
- selfhost/token added to the tree does not break bootstrap/fixpoint because the
  selfhost main package does not import it and both bootstrap stages emit it
  identically.
