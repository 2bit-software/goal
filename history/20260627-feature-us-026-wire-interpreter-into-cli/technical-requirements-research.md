# Technical Requirements / Research — US-026

## Existing seams to reuse

- `internal/parser.ParseFile(src)` → `*ast.File`.
- `internal/sema.Resolve(file)` → `*sema.Info`.
- `internal/interp.New(file, info, opts...)` with `interp.WithStdout(w)` to
  route the program's stdout effect to the command's out writer, and `Run()`
  which gates on native sema then executes `func main`.

## CLI wiring

- The prior `--engine=splice|ast` flag was removed in US-043; reintroduce
  `--engine` on `goal run` with `interp` as a value that selects the
  interpreter path. Default (no flag / `--engine=ast`) keeps the current
  transpile-and-`go run` behavior.
- The interpreter path is file-based: `<file>` is a single `.goal` file (the
  interpreter operates over one parsed `*ast.File`).
- Route the interpreter's stdout through `WithStdout(out)` so the cmd test can
  capture and assert program output.

## Test

- A `cmd/goal` test runs a sample `.goal` program (printing via `fmt.Println`)
  through the interpreter path and asserts no error (exit 0) and the expected
  stdout. Stdlib `testing` only (no testify — project constraint).
