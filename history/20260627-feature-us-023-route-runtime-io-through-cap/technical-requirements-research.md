# Technical Requirements & Research — US-023

## Current state

- `internal/cap` (US-001) defines `Capability` (Stdout, Stdin, FileRead,
  FileWrite, Net, Concurrency, Time, Env) and `CapabilitySet` with
  `Has`/`Grant` + `GrantAll()`/`DenyAll()`.
- `internal/interp/host.go` is the only place an effect reaches the OS:
  `"fmt.Println"` does `fmt.Fprintln(os.Stdout, ...)`. Sprintf/Sprint/Errorf/
  errors.New are pure (no effect). The other shimmed host calls produce values
  only.
- `Interp` is constructed via `New(file, info)`; existing tests call that exact
  2-arg form. No package outside `internal/interp` imports it yet.

## Plan

- Add to `Interp` two fields: `caps cap.CapabilitySet` and `stdout io.Writer`.
- `New` defaults `caps = cap.GrantAll()` and `stdout = os.Stdout`. Keep the
  2-arg call site working by adding variadic functional options
  (`New(file, info, opts ...Option)`).
- Add options `WithCapabilities(cap.CapabilitySet)` and `WithStdout(io.Writer)`.
- Add a capability-mediated effect gate on `Interp`, e.g.
  `emitStdout(write func(io.Writer) error) error`, that routes the write
  through `ip.stdout` only via the `cap.Stdout` authority. Under the default
  `GrantAll` it performs the write. (US-024 turns the not-granted branch into a
  loud, located, named error; here it simply does not perform the effect.)
- Route `"fmt.Println"` through that gate instead of `fmt.Fprintln(os.Stdout,
  ...)`. Because the effectful shim needs the interpreter's sink + caps, route
  it through the interpreter (evalHostCall) rather than the pure package-level
  `hostFuncs` registry.

## Constraints

- Zero third-party deps; stdlib `testing` only (no testify).
- `internal/interp` must not gain a dependency on `go/types`/`internal/typecheck`
  (US-022 gate `TestInterpHasNoGoTypesOrTypecheckDep`). `internal/cap` is
  dependency-free, so importing it is safe.
- Verify gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
