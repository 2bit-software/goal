# Plan Coverage Audit — US-009

All acceptance criteria trace to plan elements:
- AC1 (goal source importing token/ast/parser) -> 12 selfhost/sema/*.goal files.
- AC2 (transpiles + compiles, go/* pass through) -> BuildTranspiled gate.
- AC3 (existing tests pass) -> BuildAndTest behavioral gate.
- task check / build / fixpoint -> project gates.

No scope creep: plan touches only selfhost/sema, the port_test, prd.json,
progress.txt. No CRITICAL/MAJOR findings.

- MINOR: exact behavioral test-file set confirmed empirically during
  implementation (precedent US-007/US-008).

## Assumptions
- Layout/deps include lexer because the transpiled parser imports
  goal/internal/lexer even though sema does not import lexer directly.
