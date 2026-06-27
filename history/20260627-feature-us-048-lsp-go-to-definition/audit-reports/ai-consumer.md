# AI-Consumer Readiness Audit — US-048

## Findings

- The spec is implementable without guessing: it names the request
  (`textDocument/definition`), the response shape (`Location` or null), the
  reference kinds (call, enum variant, type/enum name), and every failure mode.
- Test assertions are derivable directly from the acceptance criteria (advertise
  capability; call -> func decl; variant -> variant decl; null fallbacks).
- MINOR: "AST symbol graph" is the established pattern from `symbols.go` /
  `semantictokens.go` (parse + `ast.Walk`, keyed by structural parent); an
  implementer should follow it for consistency.
- No CRITICAL or MAJOR findings.

## Assumptions

- Reuse `check.OffsetToPosition` + `rangeOf`; add a local position->offset
  helper (none exists in the tree).
- The declaration index is name-keyed (functions/methods, types/enums/sealed,
  and `Enum.Variant`).
