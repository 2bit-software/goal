# Tasks Audit

## Findings

No CRITICAL or MAJOR findings. Four tasks decompose the plan in dependency order
(lower.go facts -> emit.go prelude/funcDecl -> emit.go lowering -> tests). Each
modifies 1 file (test task included), is independently committable, and carries a
concrete verify command. Every FR is covered: FR-1 (Task 1/2), FR-2/3/4/5 (Task 3,
FR-5 emission in Task 2), all ACs (Task 4). Every plan file appears in a task.

- MINOR: Tasks 2 and 3 both touch emit.go sequentially; acceptable since they are
  one agent run and the file compiles after each (Task 2 leaves roResultClosed
  unused-but-valid until Task 3 wires returnStmt/unwrap/resultMatch).

## Assumptions

- The whole story is implemented in one agent turn (the loop's unit of work), with
  the project gates (build/vet/test) as the final acceptance — consistent with
  prior US-03x stories in this repo.
