# Tasks Audit: Buildability — US-023

- Each task compiles at its boundary: T1 is additive AST (builds alone), T2 wires
  parser against T1's nodes, T3 adds tests, T4 verifies.
- Concrete verify command per task (go build/test on the touched package).
- Interface contracts carried from the plan (AssertStmt/DocComment/Doctest
  signatures; FuncDecl.Doc). FuncMod/Mod/ModPos already exist.
- Integration points name file + function + how (newParser filter, parseFile
  collectDoc, parseStmt ASSERT case, parseDecl from/derive branch).

No CRITICAL/MAJOR findings.

## Assumptions
- parseExpr halting at a top-level comma is relied on for the assert message
  split (verified against the existing expression parser).
