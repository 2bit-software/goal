# Audit Report — US-049 spec

## Findings

- CRITICAL: none.
- MAJOR: none.
- MINOR: The spec lists struct/type-alias/enum/sealed hover support (FR-1) for
  completeness, but only the function-signature path (AC-2) is independently
  asserted by a required test. This is acceptable: the symbol graph already
  resolves those nodes, so covering them adds no risk, and AC-1 (type + doc) is
  still testable on the function case.

## Verdict

PASS. The spec is implementable as written, every acceptance criterion is
verifiable, and it stays within the established best-effort single-document LSP
contract. No implementation details leaked.

## Assumptions

- Hover content is rendered as LSP `MarkupContent` (markdown) — the standard,
  widely-supported shape; the spec does not mandate a wire format.
- "Signature" for a function = its written declaration header (modifier + `func`
  + name + params + results) sliced from source and whitespace-normalized, not
  an inferred/elaborated type.
- Doc comments are surfaced only for declarations that carry them in the AST
  (functions/methods via `FuncDecl.Doc`); enum/type nodes have no doc field, so
  their hover shows the header only.
