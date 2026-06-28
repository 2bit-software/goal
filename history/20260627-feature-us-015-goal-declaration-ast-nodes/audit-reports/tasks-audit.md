# Tasks Audit — US-015

## Coverage
- Every spec FR (FR-1..FR-5) and acceptance criterion maps to Task 1.
- Every file in the plan inventory (goal_decl.go, ast.go, walk.go, ast_test.go)
  appears in Task 1.
- No files outside the plan are referenced (no scope creep).

## Ordering
- Single task; no inter-task dependencies, so the DAG is trivially valid.
- Internal build order within the task follows the plan's topological order
  (new nodes → struct-field edits → Walk cases → test), so the package compiles
  once the task completes.

## Executability
- Task touches 4 files (within the 3-5 limit) and is a single cohesive,
  independently-committable change completable in one turn.
- Verification is concrete: `go build ./...`, `go vet ./...`,
  `go test ./... -count=1`.

## Findings
No CRITICAL or MAJOR findings. Recommend PASS.

## Assumptions
- Keeping the change as one task (rather than splitting node-defs from Walk
  cases) is correct here: splitting would leave an intermediate state where new
  nodes exist but Walk lacks their cases — still compiling, but the test (which
  needs Walk descent) can only pass once both land. One atomic task is cleaner.
