# Implementation Plan — US-023 Route runtime IO through cap

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/cap_io_test.go` | Unit tests: a printing program under the default `GrantAll` writes its expected output to a captured sink; the default interpreter grants every capability; non-effect programs are unchanged. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/interp/interp.go` | Add `caps cap.CapabilitySet` and `stdout io.Writer` fields to `Interp`. Change `New` to `New(file, info, opts ...Option)` defaulting `caps = cap.GrantAll()` and `stdout = os.Stdout`. Add the `Option` type plus `WithCapabilities` / `WithStdout`. Add the `emitStdout(write func(io.Writer) error) error` capability-mediated effect gate. |
| `internal/interp/host.go` | Route `"fmt.Println"` through the interpreter's `emitStdout` gate instead of `fmt.Fprintln(os.Stdout, ...)`. Move the effectful call out of the pure package-level `hostFuncs` registry: `evalHostCall` intercepts `fmt.Println` and calls the gate; the pure shims stay in the registry. Drop the now-unused `os` import from `host.go` if no longer referenced. |

## Package Structure

```
internal/
  cap/        # unchanged — provides Capability, CapabilitySet, GrantAll, DenyAll
  interp/
    interp.go      # Interp gains caps + stdout, New options, emitStdout gate
    host.go        # fmt.Println routed through emitStdout
    cap_io_test.go # NEW — capability-sink stdout test
```

## Dependency Graph

1. `internal/cap` (exists, no change) — the authority model.
2. `internal/interp/interp.go` — Interp fields, options, `emitStdout` gate (imports `io`, `os`, `goal/internal/cap`).
3. `internal/interp/host.go` — `evalHostCall` routes `fmt.Println` through (2)'s gate.
4. `internal/interp/cap_io_test.go` — exercises (2) + (3) end to end.

## Interface Contracts

```go
// internal/interp/interp.go
type Option func(*Interp)

func WithCapabilities(s cap.CapabilitySet) Option // sets ip.caps
func WithStdout(w io.Writer) Option               // sets ip.stdout

func New(file *ast.File, info *sema.Info, opts ...Option) *Interp
// defaults: ip.caps = cap.GrantAll(); ip.stdout = os.Stdout; then apply opts.

// emitStdout routes a stdout write through the cap.Stdout authority. Under a
// granted Stdout it performs write(ip.stdout); when Stdout is not granted it
// performs no write (the located, named denial error is US-024). Returns
// write's error.
func (ip *Interp) emitStdout(write func(io.Writer) error) error
```

```go
// internal/interp/host.go
// evalHostCall intercepts the effectful "fmt.Println" key and routes it:
//   ip.emitStdout(func(w io.Writer) error { _, err := fmt.Fprintln(w, goArgs(args)...); return err })
// The pure shims (fmt.Sprintf/Sprint/Errorf, errors.New) remain in hostFuncs.
```

## Integration Points

- `internal/interp/interp.go` `New`: defaults `caps`/`stdout`, applies options.
- `internal/interp/host.go` `evalHostCall`: after evaluating args and computing
  `key`, if `key == "fmt.Println"` route through `ip.emitStdout` and return;
  otherwise fall through to the `hostFuncs` registry lookup as today.
- `cap.Stdout` is the authority gating the stdout effect.

## Testing Strategy

- `cap_io_test.go` (package `interp`, stdlib `testing`, no testify):
  - `TestPrintlnUnderGrantAllWritesToSink`: parse+resolve a `package main` program
    whose `main` calls `fmt.Println(...)`, construct the interpreter with
    `WithStdout(&bytes.Buffer{})` (default caps = GrantAll), `Run()`, assert the
    buffer holds the expected line.
  - `TestNewDefaultsGrantAllCapabilities`: a freshly constructed interpreter has
    every `cap.Capability` granted (iterate, assert `ip.caps.Has(c)`), proving the
    default-authority requirement.
  - `TestEmitStdoutRoutesThroughConfiguredSink`: a direct `emitStdout` call writes
    to the configured sink (guards the gate seam independent of the shim).
- Existing tests keep calling `New(file, info)` unchanged (variadic options).
