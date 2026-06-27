# Technical Requirements / Research — US-007

## Approach

Add a behavioral-tier runner to `internal/corpus` that, given a transpile Case
and a Transpiler, transpiles the input and writes `Output.Go` into a freshly
created temp module (a `go.mod` plus the generated `.go` file), then shells out
to `go build` and `go vet` on that module.

## Reuse

- Temp-dir machinery pattern mirrors `cmd/goal/main.go` `writeOverlay` /
  `runGo` (os.MkdirTemp + exec.Command("go", ...)). The behavioral tier uses a
  standalone temp module rather than `-overlay`, since each case is judged in
  isolation.
- The generated Go is zero-dependency (stdlib imports only) and the module name
  is arbitrary; a minimal `go.mod` (`module goalcorpus\n\ngo 1.26\n`) suffices.
- Package name in the generated file varies (app, status, shape, ...); a single
  non-main package compiles fine in its own dir.

## Test placement

The behavioral test belongs in the external `corpus` test package alongside the
existing runner tests (cwd = internal/corpus, repoRoot = "../..", manifest at
"../../corpus/manifest.json"). Building each case is slow-ish, so the test
should be guarded with `testing.Short()` skip to stay friendly under `-short`.
