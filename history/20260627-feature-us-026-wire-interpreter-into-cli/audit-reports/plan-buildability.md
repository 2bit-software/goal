# Plan Audit: Buildability — US-026

- Dependency order valid: flag parsing → cmdRunInterp → dispatch → tests. No
  forward references.
- Interface contracts agree with the real seams:
  - `parser.ParseFile(string) (*ast.File, error)`
  - `sema.Resolve(*ast.File) *sema.Info`
  - `interp.New(*ast.File, *sema.Info, ...interp.Option) *Interp`,
    `interp.WithStdout(io.Writer) Option`, `(*Interp).Run() error`
  These are confirmed present in internal/interp and used by doctest.go.
- File paths correct: only `cmd/goal/main.go` and `cmd/goal/main_test.go` exist
  and are modified.
- Integration point specific: `run()`'s `case "run"` branches on the parsed
  engine value before the existing `cmdRun`.

No CRITICAL or MAJOR findings.

## Assumptions
- `--engine` is parsed only for `run` (not build/check), reusing the single-path
  convention; the path becomes one `.goal` file when engine=interp.
