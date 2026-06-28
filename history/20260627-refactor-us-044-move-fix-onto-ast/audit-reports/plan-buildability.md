# Plan Buildability Audit

- AST infrastructure exists and was verified by dumping the test snippets:
  FuncType.Results carries Opening/Closing parens; AssignStmt/IfStmt/ReturnStmt/
  SwitchStmt/CaseClause/StarExpr/SelectorExpr/BinaryExpr nodes carry Pos/End
  offsets; sema.Resolve populates FuncSignatures (Mode/T) and Enums. PASS
- scan.Splice + scan.Replacement remain available as pure string utilities
  (sorts, skips overlaps) — not lexing. PASS
- analyze.ZeroLit is a pure helper reusable with an AST-built typeDecls map. PASS
- Parser proven to parse all test inputs cleanly (perr=nil for each snippet). PASS

Plan is buildable with the existing packages; no new dependencies.
