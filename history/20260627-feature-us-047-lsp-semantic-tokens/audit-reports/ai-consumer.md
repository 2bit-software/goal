# Spec Audit — AI Consumer

## Findings
The spec is implementable without guessing. The LSP semantic-tokens wire format
is well-specified (see research-findings.md), and the goal lexer + AST provide
all needed positions and roles.

No CRITICAL or MAJOR findings.

- MINOR: The exact `tokenType` legend indices are an implementation choice;
  documented in the plan.

## Assumptions
- The handler follows the existing `documentSymbols` pattern (parse the open
  buffer, walk the AST) and reuses `check.OffsetToPosition` for line/char.
- Best-effort contract: empty (non-nil) result on parse failure / unknown URI.
