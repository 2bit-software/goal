# Tasks Audit — US-027

## Coverage
- Every plan file appears in a task: sema.go (T1), resolve.go (T2),
  resolve_test.go (T3).
- Every FR maps to a task: FR-1..FR-5 → T1 (shapes) + T2 (walk); FR-6 → T2/T3.
- Each AC has a verification step (build/vet/test; the parity + comma test).

## Buildability
- Ordering respects the dependency graph (types → walk → test); each task is
  independently committable and compiles after completion.
- T1 ≤1 file, T2 ≤1 file, T3 ≤1 file — all within the 3-5 file limit.
- Each task has a concrete verify command, not "check it works".

## Findings
- No CRITICAL/MAJOR. One MINOR: T2 and T3 could be one commit since the test
  proves T2; keeping them split is fine and each still compiles.

## Assumptions
- The representative test source stays within the parser's supported grammar
  (validated by running `go test`).
