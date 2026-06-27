# Plan Audit: Coverage — US-023

- FR-1 (from/derive) → parseDecl dispatch + FuncDecl.Mod/ModPos (existing) +
  TestParseFromDerive. Covered.
- FR-2 (assert) → AssertStmt node + parseAssertStmt + parseStmt case +
  TestParseAssert (incl. top-level-comma split case). Covered.
- FR-3 (doctests) → DocComment/Doctest nodes + collectDoc/extractDoctests +
  newParser/parseFile changes + TestParseDoctests. Covered.
- Walk acceptance → walk.go cases + Walk coverage assertion. Covered.
- Build/vet/test gate → standard verify commands. Covered.
- No scope creep: every plan file traces to a requirement.

No CRITICAL/MAJOR findings.

## Assumptions
- Doc attaches to the following FuncDecl only; doc before non-func decls dropped.
