# Tasks Audit (US-050)

## Coverage
- Every FR (1-5) and AC maps to a task (see task `Spec coverage` lines).
- Every plan file appears: protocol.go (T1), references.go (T2), server.go (T3),
  references_test.go (T4). No file outside the plan is referenced.

## Ordering
- DAG: T1 -> T2 -> T3 -> T4 -> T5; no cycles, no backward deps.
- Compiles at each step: T1 adds types (compiles standalone); T2 adds references.go
  using T1's types + existing helpers (compiles); T3 wires routing (compiles); T4
  adds tests; T5 is verification only.

## Executability
- Each task names concrete files, functions, and the existing code to mirror
  (definition.go refVisitor, codeaction.go WorkspaceEdit construction,
  definition_test.go helpers).
- Verify commands are concrete and runnable.

No CRITICAL/MAJOR findings. Cleared to implement.
