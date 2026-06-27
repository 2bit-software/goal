# Research — US-008

## Summary

No external research required. This story composes two patterns already proven
in the corpus package:

1. **US-005 / `RunDoctest`** establishes KindDoctest case selection. Each
   doctest case shares its Input with a transpile twin; transpiling yields
   `Output.Go` (the package under test, e.g. `package mathx`) and `Output.Test`
   (the emitted `_test.go` doctest sidecar).
2. **US-007 / `RunCompile`** establishes the isolated-temp-module machinery:
   `os.MkdirTemp`, a minimal `module goalcorpus` / `go 1.26` go.mod, write the
   generated `.go` file, then `exec.Command("go", verb, "./...")` with
   `cmd.Dir = tmp` and `CombinedOutput()` for diagnostics.

US-008 = write BOTH files (package + sidecar) into the temp module and run
`go test ./...` instead of `go build`/`go vet`.

## Confidence

High — both seams exist and are unit-tested in the same package.

## Open Questions

None. The 4 feature-11 doctest cases are the doctest-bearing set per the
manifest.

## Next Steps

Implement `RunDoctestExec` + a `-short`-skipped whole-corpus test.
