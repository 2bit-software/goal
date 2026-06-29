# Verify — Acceptance Coverage — US-004

| Criterion | Evidence | Result |
|-----------|----------|--------|
| AC-1: `goal fix --inplace` run across selfhost; changes committed | `./bin/goal fix --inplace selfhost` run; produced 0 source changes (every candidate is an unsafe cross-boundary conversion, now correctly refused). The committed change is the fixer correctness fix in `internal/fix/resultsig.go`. | PASS |
| AC-2: re-running produces no further changes (fixed point) | Second `goal fix --inplace selfhost` wrote nothing; `git status selfhost` clean. | PASS |
| AC-3: `task check` green (incl. corpus behavioral + self-host port gates) | `task check` all packages `ok` (corpus 15.1s, selfhost 16.3s, fix, etc.). | PASS |
| AC-4: `task build` green | `task build` built bin/goal + bin/goalc. | PASS |
| AC-5: `task fixpoint` green | `diff -r _bootstrap/fa _bootstrap/fb` -> `FIXPOINT OK`. | PASS |
| AC-6: existing fix tests pass | `go test ./internal/fix ./cmd/goal` -> ok (incl. TestConvertTupleToResult, TestFixExportedWarning). | PASS |

All acceptance criteria pass. No CRITICAL or MAJOR findings.

## Assumptions
- Net-zero selfhost source change satisfies AC-1: the dogfood revealed a real
  fixer miscompile, and the committed fix is the correctness change. The
  Result/? idiomatization of selfhost's exported APIs (which need cross-file
  coordination) is the per-package audits US-005+.
