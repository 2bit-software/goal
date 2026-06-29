# Implementation Tasks — US-007 Port ast package to goal

## Task 1: Copy ast sources to selfhost/ast as goal
**Status**: completed
**Files**: selfhost/ast/ast.goal, selfhost/ast/walk.goal, selfhost/ast/goal_decl.goal, selfhost/ast/goal_expr.goal, selfhost/ast/goal_stmt.goal
**Depends on**: (none)
**Spec coverage**: FR-1
**Verify**: files exist; `grep -nE '\b(match|enum|assert)\b'` shows only the `len("assert")` string literal

### Instructions
- Copy internal/ast/{ast,walk,goal_decl,goal_expr,goal_stmt}.go verbatim to
  selfhost/ast/<same>.goal. Go superset == valid goal.
- Do NOT copy dump.go (reflection-driven debug Sexpr; off compile path,
  unreferenced).
- No edits needed: the only reserved-word hit is the string literal
  `len("assert")` in goal_stmt.go.

## Task 2: Add TestPortedAstPackage port test
**Status**: completed
**Files**: internal/selfhost/port_test.go
**Depends on**: Task 1
**Spec coverage**: FR-2, FR-3
**Verify**: `go test ./internal/selfhost -run TestPortedAstPackage`

### Instructions
- Mirror TestPortedLexerPackage. Discover ../../selfhost/token and
  ../../selfhost/ast; assert package names "token" and "ast".
- BuildTranspiled layout {"internal/token": tokenPkg, "internal/ast": astPkg}.
- BuildAndTest("internal/ast", astPkg, []string{"../ast/ast_test.go"},
  deps={"internal/token": tokenPkg}).

## Task 3: Verify project gates and finalize
**Status**: completed
**Files**: prd.json, progress.txt
**Depends on**: Task 2
**Spec coverage**: all
**Verify**: `task check`, `task build`, `task fixpoint`
