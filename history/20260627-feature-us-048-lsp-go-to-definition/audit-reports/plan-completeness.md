# Plan Coverage Audit — US-048

## Findings

- Every FR maps to a plan element: FR-1 -> capability advertise; FR-2 ->
  CallExpr ref; FR-3 -> VariantLit/VariantPattern/SelectorExpr ref; FR-4 ->
  type-position + selector-enum ref; FR-5 -> nil returns. No requirement is
  orphaned and no plan element lacks a requirement.
- Every acceptance criterion has a testing strategy in the Testing section.
- No CRITICAL or MAJOR findings.

## Assumptions

- Returned range is the declaration name's range (via `rangeOf`).
- Single-document resolution (open buffer's AST).
