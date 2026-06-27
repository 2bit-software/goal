# Research Findings — US-028

## Summary

US-028 is a conformance GATE that composes two already-built, in-repo seams; no
external research or new runtime mechanics are required.

## Verified seams (read from source)

- **Interpreter run path**: `internal/interp.New(file, info, opts...)` +
  `(*Interp).Run()` (interp.go:153, :342). `Run()` finds and executes
  `func main`. `interp.WithStdout(w)` routes the program's stdout effect through
  the capability sink to `w` (US-023), and `New` defaults to `cap.GrantAll()`.
  Proven runnable for a fmt.Println-printing enum+match program by the US-026
  CLI test `TestRunInterpEngineExecutesMain` (output "green").

- **Transpile path**: `internal/backend.Transpile(src) (pipeline.Output, error)`
  (backend.go:78) parses, sema-resolves, emits, and go/format-normalizes to
  `Output.Go` (`internal/pipeline.Output{Go, Test}`).

- **Temp-module + toolchain pattern**: `internal/corpus.RunDoctestExec`
  (doctest_behavior_runner.go) writes a minimal `module ...\n\ngo 1.26\n` go.mod
  plus the generated Go into an `os.MkdirTemp` dir and invokes the `go`
  toolchain via `exec.Command` with `cmd.Dir = tmp`. For a `func main` program
  the equivalent is `go run .` capturing `CombinedOutput`/stdout.

## Import-cycle check

`internal/corpus` already imports `internal/interp`. It may also import
`internal/backend`: `go list -deps goal/internal/backend` shows no dependency on
corpus, so corpus -> backend introduces no cycle.

## Sample program choice

Reuse the US-026 fixture shape: an `enum Color { Red; Green }` + a value-position
`match` returning a string + `func main` printing it. This exercises a GENUINE
goal construct (enum + value-position match — one of the two non-Go runtime
mechanics) so the no-op upgrade is meaningful, not a trivial pure-Go program.
Expected observable output: `green\n` under both paths.

## Confidence

High — every seam was read directly from the current source and is already
exercised by existing green tests.
