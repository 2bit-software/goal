# AI-Consumer Audit — US-029

The diagnostics are agent-facing. Parity with the legacy messages keeps them actionable:
- Non-exhaustive Error names the missing variants in declaration order and tells the
  agent exactly how to fix it ("handle ..., or add a `_` rest-arm to dismiss the rest").
- Deferred Warning names the unresolved enum rather than silently assuming completeness.

No CRITICAL/MAJOR findings.

## Assumptions

Same as completeness.md.
