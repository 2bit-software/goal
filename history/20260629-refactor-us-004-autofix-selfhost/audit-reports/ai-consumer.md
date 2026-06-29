# AI-Consumer Readiness Audit — US-004

## Findings

- The fix is precisely specified in technical-requirements-research.md: the two
  conservative refusal rules (exported; non-collapsible reference) and the three
  concrete selfhost miscompiles they address. An implementer can write test
  assertions directly from the existing internal/fix test suite.
- Acceptance criteria map 1:1 to commands (`goal fix`, `task check/build/fixpoint`,
  `go test ./internal/fix ./cmd/goal`). No guessing required.

No CRITICAL or MAJOR findings.

## Assumptions

- Existing fix tests encode the required-conversion cases (unexported function
  with zero in-file callers, or with a single collapsible-propagation call site);
  the new refusal rules must keep these green. Verified by inspection before
  implementation.
