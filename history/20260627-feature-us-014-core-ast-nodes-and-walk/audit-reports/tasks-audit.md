# Tasks Audit — US-014

## Coverage
- FR-1..FR-4 → Task 1 (node types). FR-5 → Task 2 (Walk). Acceptance test → Task 3.
- Plan file inventory (ast.go, walk.go, ast_test.go) all appear. No extra files.

## Ordering
- Task 1 → Task 2 → Task 3: valid DAG, no forward deps. Note: after Task 1 alone
  the package compiles (types only); Task 2 adds Walk; Task 3 is test-only.
  Codebase compiles after each non-test task.

## Executability
- Each task names concrete files, a verify command, and references go/ast as the
  shape model. Each modifies a single file. Instructions are unambiguous.

## Sizing
- Three right-sized tasks; none trivial, none over 5 files. Task 1 is the largest
  (the node catalog) but is a single cohesive file completable in one turn.

## Findings
None CRITICAL / None MAJOR. Ready to implement.

## Assumptions
- go/ast is the field-shape reference for nodes (trimmed to goal's subset).
- Tasks 1+2 are committed together with the test (Task 3) as one story commit.
