# Tasks Audit: Coverage — US-023

- FR-1/FR-2/FR-3 each appear in Task 1 (nodes) and Task 2 (parser); tests in
  Task 3; verify in Task 4. Full coverage.
- Every plan file is assigned: ast/goal_stmt.go, ast/ast.go, ast/walk.go (T1);
  parser/goal_stmt.go, parser/parser.go (T2); parser/goal_stmt_test.go (T3).
- Each task is independently committable and ≤3 files. Ordering respects the
  dependency graph (AST → parser → tests → verify).

No CRITICAL/MAJOR findings.

## Assumptions
- Walk coverage assertion lives in the new parser test (T3) rather than
  ast_test.go; either satisfies the criterion.
