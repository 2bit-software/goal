# Tasks Audit — US-016

## Coverage
- All FRs covered: FR-1..FR-5 by Task 1 (node decls), FR-6 by Task 2 (Walk),
  AC 1/2/3 by Task 3 (test), AC 4 by Task 4 (full verify).
- All plan-inventory files appear in tasks: goal_expr.go (T1), walk.go (T2),
  ast_test.go (T3). No out-of-plan files referenced.

## Ordering
- Valid DAG: T1 -> T2 -> T3 -> T4. Codebase compiles after each: T1 adds
  self-contained node types; T2 adds switch cases referencing them; T3 adds a
  test; T4 is verify-only.

## Executability
- Each task has concrete instructions referencing specific existing patterns
  (goal_decl.go, walk.go helpers, TestWalkGoalDeclChildren) and a runnable verify
  command. Each touches <= 1 source file.

## Sizing
- Tasks are right-sized for one turn; none trivial, none oversized.

## Findings
- None CRITICAL / None MAJOR.
- MINOR: Task 4 modifies no files (verify gate only); acceptable as an explicit
  final acceptance step.

## Assumptions
- MatchArm.Body modeled as `Node` (walked via Walk, nil-guarded), not Expr/Stmt.
- VariantPattern carries a single optional payload Binding.
- The distinct-type assertion uses `%T` string comparison (a standard,
  dependency-free approach).
