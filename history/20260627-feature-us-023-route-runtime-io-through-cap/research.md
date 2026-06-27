# Research — US-023 Route runtime IO through cap

This is an internal, self-contained wiring change against code already in this
repo; no external/library research was required. Findings come from reading the
existing packages.

## Findings

1. `internal/cap` already provides the authority model (US-001): a value-type
   `CapabilitySet` bitset with `Has(c)`, `Grant(c)`, `GrantAll()`, `DenyAll()`.
   It is dependency-free, so `internal/interp` can import it without breaching
   the US-022 no-`go/types`/no-`typecheck` dependency gate.

2. The ONLY interpreter host effect that reaches the OS today is the
   `"fmt.Println"` shim in `internal/interp/host.go`, which does
   `fmt.Fprintln(os.Stdout, ...)`. The other shims (`fmt.Sprintf`, `fmt.Sprint`,
   `fmt.Errorf`, `errors.New`) are pure value producers — no effect to route.
   `time`/`env` reads named in the story are not yet shimmed, so the seam must
   exist for them but there is no current call site to convert.

3. The `hostFuncs` registry is a package-level `map[string]hostFunc` of pure
   `func(args []Value) ([]Value, error)`. An effectful shim needs the
   interpreter's configurable sink + capability set, which a package-level pure
   function cannot reach. Idiomatic Go answer: intercept the effectful call in
   `evalHostCall` (a method on `*Interp`) and route it through an
   interpreter-held, capability-mediated emit gate. The pure shims stay in the
   registry.

4. `Interp` is built via `New(file, info)` and every existing test uses that
   exact 2-arg form; nothing outside `internal/interp` imports the package.
   Variadic functional options (`New(file, info, opts ...Option)`) add the
   stdout sink + capability injection without breaking any call site — the
   standard Go pattern for optional construction config.

## Decision

Add `caps cap.CapabilitySet` + `stdout io.Writer` to `Interp`, default them to
`GrantAll()` + `os.Stdout` in `New`, expose `WithCapabilities`/`WithStdout`
options, add an `emitStdout` capability gate, and route `fmt.Println` through
it. Confidence: High. Denial-to-error is deferred to US-024.
