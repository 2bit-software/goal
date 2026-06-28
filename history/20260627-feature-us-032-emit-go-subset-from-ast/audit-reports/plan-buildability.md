# Plan Audit 2: Buildability — US-032

## Findings

### No CRITICAL or MAJOR
- Dependency order valid: emitter change → fixture → tests; no forward refs.
- Interface contracts concrete: `switchStmt(*ast.SwitchStmt)` /
  `caseClause(*ast.CaseClause)` use real AST field names (`Init`, `Tag`, `Body`,
  `List`) verified against internal/ast/ast.go.
- File paths verified: internal/backend/emit.go and backend_test.go exist;
  testdata/plain_full.goal is a new sibling of the existing plain.goal.
- Integration point specific: emit.go `stmt` type switch gains
  `case *ast.SwitchStmt`, dispatching `switchStmt`, which iterates the body's
  `*ast.CaseClause` elements directly.

### MINOR — gofmt normalizes layout
Emitter need only produce token-correct Go (e.g. `case e1, e2 :` spacing
irrelevant); `GoFormatter` fixes layout. The plan already states the format-once
discipline, so the implementer won't hand-format.

## Assumptions
- `corpus.RunCompile` and `corpus.TranspilerFunc` signatures are unchanged from
  US-026 (verified in the existing backend_test.go), so the new test compiles by
  reusing the same call shape.
