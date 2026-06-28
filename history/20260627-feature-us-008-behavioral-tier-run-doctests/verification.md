# Verification — US-008

## Acceptance Criteria

- [x] For each case that has a doctest sidecar, the runner writes both files to a
      temp module and runs `go test`. → `RunDoctestExec` writes case.go +
      case_test.go + go.mod and execs `go test ./...`.
- [x] A test asserts every doctest-bearing case passes go test in its temp
      module. → `TestDoctestExecRunner` drives all 4 KindDoctest cases; all PASS.
- [x] Loud empty-corpus guard → `t.Fatalf` when zero ran.

## Gate Results

- `go build ./...` — OK
- `go vet ./...` — OK
- `go test ./... -count=1` — all packages ok (internal/corpus 7.7s incl. new tier)
- `go test ./internal/corpus/ -run TestDoctestExecRunner -v` — 4/4 subtests PASS

## Conclusion

Implementation satisfies the spec. Source tree untouched (temp module only).
