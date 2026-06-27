# Implementation Plan — US-028 Gate: script-to-module no-op

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/corpus/script_module_gate_test.go` | The no-op gate test: run a sample goscript program under the interpreter, transpile+build+run the same source as a Go module, assert identical stdout. |

### Modified Files
None. US-028 is a conformance gate composing existing seams; no production code
changes are required (mirrors US-027, which added a gate test only).

## Package Structure

```
internal/corpus/
  script_module_gate_test.go   (new — package corpus_test)
```

The test lives in an EXTERNAL test package `corpus_test` so it can import
`internal/backend` and `internal/interp` alongside `internal/corpus` without any
risk of an internal-package import cycle (backend does not import corpus; interp
does not import corpus).

## Dependency Graph

1. Sample program source constant (no dependencies).
2. Interpreter-run helper: parse -> sema.Resolve -> interp.New(WithStdout) ->
   Run; capture buffer. Depends on 1.
3. Transpile-build-run helper: backend.Transpile -> temp module (go.mod +
   case.go) -> `go run .`; capture stdout. Depends on 1.
4. Gate test asserting (2) == (3) and non-empty. Depends on 2 and 3.

## Interface Contracts

Existing seams consumed (no new exported API):

```go
// internal/parser
func ParseFile(src string) (*ast.File, error)

// internal/sema
func Resolve(file *ast.File) *sema.Info

// internal/interp
func New(file *ast.File, info *sema.Info, opts ...Option) *Interp
func WithStdout(w io.Writer) Option
func (ip *Interp) Run() error

// internal/backend
func Transpile(src string) (pipeline.Output, error) // Output.Go is the module source
```

Test-local helpers (unexported, in the test file):

```go
func runUnderInterp(t *testing.T, src string) string          // captured stdout
func runAsGoModule(t *testing.T, goSrc string) string         // captured stdout from `go run .`
```

## Integration Points

- Interpreter path: `interp.New(file, info, interp.WithStdout(&buf))` then
  `ip.Run()` — identical wiring to cmd/goal `cmdRunInterp` and
  TestRunInterpEngineExecutesMain.
- Backend path: `backend.Transpile(src)` -> `out.Go`; write `go.mod`
  (`module goalscript\n\ngo 1.26\n`) + `case.go` into `os.MkdirTemp`; run
  `exec.Command("go", "run", ".")` with `cmd.Dir = tmp` — the same temp-module +
  toolchain pattern as `internal/corpus.RunDoctestExec`.

## Testing Strategy

Single gate test `TestScriptToModuleNoOp` in `package corpus_test`:
- Sample source: `enum Color { Red; Green }` + value-position `match` returning a
  string + `func main` printing it via `fmt.Println` (a genuine goal construct,
  not pure Go). Expected output: `green`.
- Assert `runUnderInterp(src) == runAsGoModule(transpile(src).Go)` and that the
  shared output is non-empty and equals the expected `green`.
- Failure modes (transpile error, build/run error, mismatch) each `t.Fatalf`
  with both outputs / the toolchain combined output, so a divergence is loud
  (FR-4).
- stdlib `testing` only; no testify.

## Spec Traceability

- FR-1 -> `runUnderInterp` + sample program.
- FR-2 -> `runAsGoModule` (transpile + `go run`).
- FR-3 -> the equality assertion.
- FR-4 -> `t.Fatalf` on each failure path reporting both outputs.
