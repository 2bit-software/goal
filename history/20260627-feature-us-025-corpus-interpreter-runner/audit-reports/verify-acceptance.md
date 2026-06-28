# Verify — Acceptance (US-025)

Verified against business-spec.md acceptance criteria. All green.

| Criterion | Result |
|-----------|--------|
| corpus exposes RunInterp; loads a case, runs through interpreter, compares observable behavior | PASS — `internal/corpus/interp_runner.go`; `TestInterpRunner` drives every manifest doctest case |
| every doctest corpus case passes through RunInterp | PASS — `go test ./internal/corpus -run TestInterpRunner` green (4 doctest cases) |
| a mutated wrong expected makes RunInterp fail loudly | PASS — `TestInterpRunnerMutatedExpectedFails`, `TestRunDoctestsReportsMismatch` |
| a wrong-kind case is refused | PASS — `TestInterpRunnerWrongKind` |
| no go/types / depth-checker dependency added | PASS — `TestInterpHasNoGoTypesOrTypecheckDep` still green |

## Full gate

- `go build ./...` — clean
- `go vet ./...` — clean
- `go test ./... -count=1` — all packages ok

## Assumptions

- Observable behavior for a doctest = each `>>>` expression's `Value.String()`
  rendering compared to the trimmed expected line(s); this matches the Go doctest
  tier's `got != want` assertion spelling.
- A doctest-kind case with zero `>>>` examples is treated as a loud failure
  (`ran == 0` error), not a vacuous pass — consistent with the corpus runners that
  fatal on zero cases.
