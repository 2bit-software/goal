# Completeness Audit — US-048

## Findings

- MINOR: The spec does not state which range of the declaration is returned
  (whole declaration vs. name only). Resolved by convention: the declaration's
  name (selection) range, matching `documentSymbol`'s `SelectionRange`.
- MINOR: Behavior when multiple declarations share a name (e.g. a method and a
  function) is unspecified. Acceptable: last-writer-wins in the index; not
  exercised by the corpus or AC.
- No CRITICAL or MAJOR findings. The acceptance criteria are concrete and
  independently testable.

## Assumptions

- Definition is single-document (the open buffer's AST), not workspace-wide.
- Returned location range is the declaration's name span.
- Local variables, parameters, and qualified/imported symbols are out of scope.
