# Tasks Audit — US-017

## Coverage
- FR-1..FR-5 + same-E + non-variant refusal all map to Task 2 (impl) and Task 3
  (tests). Every plan file (interp.go, eval.go, question_test.go, prd.json,
  progress.txt) appears in a task. No out-of-plan files referenced.

## Ordering
- Task 1 (fnStack) -> Task 2 (evalUnwrap reads curSig) -> Task 3 (tests) ->
  Task 4 (verify/mark/log). Valid DAG, no forward deps. Codebase compiles after
  Task 1 (field + helpers, unused-but-referenced via push), after Task 2
  (UnwrapExpr handled), and Task 3 adds tests only.

## Executability
- Each task names concrete files (<= 1-2 each) and a concrete verify command.
  Task 1 verify `go build`; Task 2 `go build && go vet`; Task 3 targeted
  `go test -run TestQuestion`; Task 4 full gates + US-022 dependency check.

## Findings
None CRITICAL/MAJOR. Ready to implement.

## Assumptions
- Task 1 leaves `fnStack` referenced (push/pop) so the build stays clean even
  before Task 2 consumes `curSig`.
