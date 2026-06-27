# Plan Audit: Buildability — US-028

## Findings

No CRITICAL or MAJOR findings.

- Dependency order is valid (sample -> interp helper / transpile helper -> gate
  assertion); no forward references.
- Interface contracts match the actual current signatures (verified against
  source): `parser.ParseFile`, `sema.Resolve`, `interp.New`/`WithStdout`/`Run`,
  `backend.Transpile` returning `pipeline.Output{Go, Test}`.
- File path `internal/corpus/script_module_gate_test.go` does not conflict with
  any existing file.
- External `package corpus_test` avoids any import cycle (backend and interp do
  not import corpus).
- Integration points name exact files/functions and the temp-module pattern is
  copied from a proven runner (RunDoctestExec).

### MINOR-1
The plan should confirm at implementation time that `backend.Transpile`'s
`Output.Go` for an enum+match `func main` program builds standalone (no doctest
sidecar needed). This is verified by actually running the test; if the emitter
needs a prelude file it is emitted into `Output.Go` already (single-file output
for file-mode transpile).

## Assumptions

- The `go` toolchain is available in the test environment (same requirement as
  the existing RunDoctestExec behavioral tier).
- `go run .` (rather than `go build` + exec) is an acceptable way to produce the
  binary's observable output.
