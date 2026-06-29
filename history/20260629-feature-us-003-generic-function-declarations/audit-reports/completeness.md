# Completeness Audit

## Findings

- MINOR: The spec does not specify behavior for `[T, U any]` shared-constraint
  lists, but the existing `parseTypeParams` already handles them, so no extra
  requirement is needed.
- MINOR: Generic methods are correctly placed Out of Scope (Go disallows them).

No CRITICAL or MAJOR findings. Spec is implementable.

## Assumptions

- Reuse of the existing `atTypeParams`/`parseTypeParams` helpers (already used
  by TypeSpec) rather than new parsing logic.
- Type params are parsed only when no receiver is present (no generic methods).
