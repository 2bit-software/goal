# Technical Requirements & Research — US-024

## Current seam

`internal/interp/interp.go` `emitStdout(write func(io.Writer) error) error` is
the single capability gate every stdout effect flows through. TODAY its
not-granted branch performs NO write and returns `nil` (a silent skip — the
US-023 placeholder). US-024 turns that branch into a located, named refusal.

`emitStdout` is called from `internal/interp/host.go` `evalHostCall` for the
`fmt.Println` interception — that call site HAS the source position
(`sel.Pos()`) and the capability being exercised (`cap.Stdout`).

## Approach

- Define a named capability-denied error in `internal/interp` carrying the
  denied `cap.Capability` and a source position, with a descriptive
  `interp: <pos>: capability denied: <Cap>` message. Expose it so a test (and a
  host) can match it (`errors.As` / sentinel field).
- Thread the source position + capability into `emitStdout` so the refusal is
  LOCATED and NAMED. Signature becomes
  `emitStdout(pos token.Pos, write func(io.Writer) error) error` (or carry a
  capability arg if multiple caps will share the gate later — Stdout is the only
  routed one today, so the gate is Stdout-specific).
- On `!caps.Has(cap.Stdout)`: return the capability error WITHOUT calling
  `write` (nothing is written). On grant: unchanged behavior.

## Constraints

- `internal/cap` is dependency-free; importing it does not breach the US-022
  dependency gate (`go list -deps internal/interp` must stay free of go/types
  and internal/typecheck). `cap` is already imported.
- Zero external deps; tests use stdlib `testing` only (no testify).
- Verify gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
