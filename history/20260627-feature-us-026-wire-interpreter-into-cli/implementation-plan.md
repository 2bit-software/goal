# Implementation Plan — US-026 Wire interpreter into the CLI

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| (none) | All changes live in cmd/goal. |

### Modified Files
| File | Changes |
|------|---------|
| `cmd/goal/main.go` | Parse a new `--engine` flag for `run`; when `--engine=interp`, dispatch to a new `cmdRunInterp` that parses + sema-resolves + interprets a single `.goal` file, routing stdout to `out`. Add `goal/internal/interp`, `goal/internal/parser`, `goal/internal/sema` imports. Document the `--engine` flag in `guideCommands` for `run`. |
| `cmd/goal/main_test.go` | Add a test running a sample `.goal` program through `--engine=interp` asserting nil error (exit 0) and expected stdout; plus an unknown-engine error case and a no-main error case. |

## Package Structure

No new packages. The interpreter (`internal/interp`), parser, and sema already
exist and are imported by `cmd/goal/main.go`.

## Dependency Graph

1. Add `--engine` parsing on the `run` subcommand (a small `parseRunFlags` or an
   extension of the run dispatch) — foundation.
2. Add `cmdRunInterp(file, out, errOut)` using parser → sema → interp.New(...,
   WithStdout(out)) → Run() — depends on 1.
3. Wire dispatch in `run()` for `cmd == "run"` to branch on the engine — depends
   on 1 & 2.
4. Tests — depend on 3.

## Design Notes

- `--engine` default is `ast` (current transpile-and-`go run` behavior). Only
  `interp` selects the interpreter; any other value is a descriptive error.
- For `--engine=interp` the path argument is a single `.goal` file (interp is
  single-`*ast.File`). If the path is a directory or not a `.goal` file, return a
  descriptive error.
- The interpreter run uses full authority (the `interp.New` default,
  `cap.GrantAll`) and `WithStdout(out)` so program output reaches the command's
  stdout and tests can capture it.
- Errors from parse / sema-gate / no-main / runtime propagate as the command's
  returned error; `main()` already prints + exits 1.

## Test Strategy

- `TestRunInterpEngineExecutesMain`: a `fmt.Println`-printing program run via
  `run([]string{"run", "--engine=interp", file}, &out, &errOut)` → nil error,
  trimmed stdout equals expected.
- `TestRunInterpUnknownEngineRejected`: `--engine=bogus` → non-nil error.
- `TestRunInterpNoMain`: a program without `func main` → non-nil error.
- Stdlib `testing` only (no testify).
