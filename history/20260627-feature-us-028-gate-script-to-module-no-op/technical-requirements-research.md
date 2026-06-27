# Technical Requirements / Research — US-028

## Existing seams to compose (no new production code expected)

- `internal/interp.New(file, info, interp.WithStdout(buf))` + `(*Interp).Run()`
  executes `func main` in-process and routes stdout through the capability sink
  (US-023/US-026). This is the interpreter run path.
- `internal/backend.Transpile(src) (pipeline.Output, error)` parses, sema-
  resolves, emits, and go/format-normalizes the program to `Output.Go`
  (the AST backend). This is the transpile path.
- `internal/corpus.RunDoctestExec` shows the temp-module pattern: write a
  minimal `go.mod` + the generated Go into an isolated `os.MkdirTemp` dir and
  invoke the `go` toolchain (`go test` there; here `go run .`).

## Plan

Add a gate test in `internal/corpus` that:
1. Holds a sample goscript program with `func main` printing via fmt (an enum +
   value-position `match` proves a real goal construct upgrades as a no-op).
2. Runs it under the interpreter (`interp.New` + `Run`, capturing stdout).
3. Transpiles it via `backend.Transpile`, writes `go.mod` + `case.go` into a
   temp module, and `go run .`, capturing stdout.
4. Asserts the two outputs are byte-equal.

## Notes

- `internal/corpus` may import `internal/backend` (backend does not import
  corpus — no cycle) and already imports `internal/interp`.
- Zero-dependency, stdlib `testing` only (no testify) per the project constraint.
