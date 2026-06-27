# Implementation Tasks — US-023 Route runtime IO through cap

## Task 1: Add capability set + stdout sink + emit gate to Interp
**Status**: completed
**Files**: `internal/interp/interp.go`
**Depends on**: (none)
**Spec coverage**: FR-1 (effects capability-mediated, the gate), FR-2 (default
authority full), FR-3 (configurable sink)
**Verify**: `go build ./...`

### Instructions
- Add imports `io`, `os`, and `goal/internal/cap` to interp.go.
- Add fields to `Interp`: `caps cap.CapabilitySet` and `stdout io.Writer`.
- Define `type Option func(*Interp)` and constructors:
  - `WithCapabilities(s cap.CapabilitySet) Option` → sets `ip.caps`.
  - `WithStdout(w io.Writer) Option` → sets `ip.stdout`.
- Change `New(file *ast.File, info *sema.Info, opts ...Option) *Interp`: after
  building `ip`, set defaults `ip.caps = cap.GrantAll()` and `ip.stdout =
  os.Stdout`, then `for _, opt := range opts { opt(ip) }` before returning.
  Keep the existing registerImports/registerFuncs/registerMethods calls.
- Add `func (ip *Interp) emitStdout(write func(io.Writer) error) error`: when
  `ip.caps.Has(cap.Stdout)` perform `return write(ip.stdout)`; otherwise return
  nil and perform no write (the located, named denial error is US-024). Document
  that contract in the doc comment.

## Task 2: Route fmt.Println through the cap gate
**Status**: completed
**Files**: `internal/interp/host.go`
**Depends on**: Task 1
**Spec coverage**: FR-1 (the one existing effect site)
**Verify**: `go build ./... && go vet ./...`

### Instructions
- Remove the `"fmt.Println"` entry from the package-level `hostFuncs` registry
  (it cannot reach the interpreter sink/caps).
- In `evalHostCall`, after computing `key` and evaluating `args`, intercept the
  effectful symbol: `if key == "fmt.Println" { return nil, ip.emitStdout(func(w
  io.Writer) error { _, err := fmt.Fprintln(w, goArgs(args)...); return err }) }`
  placed before the `hostFuncs[key]` lookup, so the unresolved-symbol refusal
  path is unchanged for everything else.
- Add the `io` import; drop `os` from host.go if it is no longer referenced
  there (it was only used by the removed Println shim). Keep the doc comment in
  sync (the file header note about US-023 routing).

## Task 3: Capability-sink stdout tests
**Status**: completed
**Files**: `internal/interp/cap_io_test.go` (new)
**Depends on**: Task 2
**Spec coverage**: AC "print under GrantAll captured through sink", FR-2, FR-3
**Verify**: `go test ./internal/interp/ -count=1`

### Instructions
- Package `interp`, stdlib `testing` + `bytes` + `strings`, no testify. Reuse
  the existing test helper that parses + sema-resolves a program (see
  host_test.go `newInterp`); for the sink test construct the interpreter
  directly with `New(file, sema.Resolve(file), WithStdout(buf))`.
- `TestPrintlnUnderGrantAllWritesToSink`: program `package main` with
  `import "fmt"` and `func main() { fmt.Println("hello", 42) }`; run with a
  `*bytes.Buffer` sink; assert buffer == "hello 42\n".
- `TestNewDefaultsGrantAllCapabilities`: build an interpreter and assert
  `ip.caps.Has(c)` is true for every `cap.Capability` (iterate the known set
  Stdout..Env), proving FR-2.
- `TestEmitStdoutRoutesThroughConfiguredSink`: call `ip.emitStdout` directly
  with a closure writing a sentinel to the sink; assert it lands in the buffer
  (guards the gate independent of the shim).
