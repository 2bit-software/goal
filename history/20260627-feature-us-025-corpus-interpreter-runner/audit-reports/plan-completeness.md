# Plan Audit — Coverage (US-025)

## Coverage map

| Spec item | Plan element |
|-----------|--------------|
| FR-1 interpreter corpus runner | `internal/corpus/interp_runner.go` `RunInterp` |
| FR-2 doctest observable-behavior comparison | `interp.RunDoctests` evaluates each `>>>` expr, compares `Value.String()` to expected |
| FR-3 loud, case-identified failures | `RunInterp` wraps failures/errors with case ID, func, input, expected, got |
| FR-4 wrong-kind / empty-case refusal | `RunInterp` kind guard + `ran == 0` loud failure |
| FR-5 behavioral parity oracle | comparison via `Value.String()` (Go-literal spelling) |
| AC: manifest doctest cases pass | `interp_runner_test.go` iterates `Kind==KindDoctest` |
| AC: mutated expected fails | temp-file mutated-doctest test |
| AC: wrong-kind refused | in-memory wrong-kind Case test |
| AC: no go/types dependency | existing interp dep gate holds; no new go/types edge |

No CRITICAL/MAJOR findings. No scope creep (no production changes to existing files).

## Assumptions

- `RunDoctests` returns `(failures []DoctestFailure, ran int, err error)` so the
  runner can distinguish "all examples passed" from "no examples at all".
