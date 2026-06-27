# Plan Audit — Buildability

## Findings

No CRITICAL findings.
No MAJOR findings.

### Checks

- Dependency order is valid: type+signature change (1) -> call-site update (2)
  -> tests (3); no forward references.
- Interface contracts agree: `emitStdout(pos token.Pos, write func(io.Writer) error) error`
  is called from host.go with `sel.Pos()` (a `token.Pos`). `token` and `cap`
  are already imported in interp.go; `ast` is already imported in host.go.
- File paths verified against the tree: `internal/interp/interp.go`,
  `internal/interp/host.go` exist; `cap_deny_test.go` is a new sibling of the
  existing `cap_io_test.go`.
- Compiles at each step: after step 1 the lone caller in host.go would fail to
  compile until step 2, but steps 1+2 are a single atomic edit pair before any
  build/verify — acceptable for a 2-line signature threading.
- The US-022 dependency gate holds: `cap` and `token` are dependency-free w.r.t.
  go/types / internal/typecheck.

## Assumptions

- `emitStdout` keeps a single position parameter (Stdout is the only routed
  effect today); a future multi-capability gate would add a `cap.Capability`
  parameter, out of scope now.
