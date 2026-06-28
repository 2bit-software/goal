# Plan Audit: Buildability — US-023

- Dependency order valid: AST nodes → Walk → parser → tests. No forward refs.
- Interface contracts concrete (actual Go signatures). FuncDecl.Mod/ModPos and
  FuncMod constants already exist (verified in internal/ast/goal_decl.go); the
  plan only adds FuncDecl.Doc and new nodes.
- File paths verified against the tree: internal/ast/{ast,walk,goal_*}.go and
  internal/parser/{parser,goal_*}.go exist; new goal_stmt.go files do not collide.
- Integration points specify file + function + how (newParser filter change,
  parseFile collectDoc, parseStmt ASSERT case, parseDecl from/derive branch).
- Compiles incrementally: adding nodes + fields is additive; keyed struct
  literals keep FuncDecl additions safe (ast_test is keyed).

Risk noted (MINOR): keeping DOC_COMMENT tokens in the stream could surprise a
loop that previously never saw them. Mitigation in plan: defensive skip in
parseBlock; DOC_COMMENT otherwise only appears at top level in the corpus.

No CRITICAL/MAJOR findings.

## Assumptions
- internal/parser stays import-limited to lexer/token/ast; tests stay internal.
- extractDoctests strips the `///` prefix + one leading space per line.
