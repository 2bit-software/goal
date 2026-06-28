# Research — US-024 Enforce capability denial

## Summary

The denial seam already exists; US-024 only turns its placeholder branch into a
loud refusal. No external research needed — this is an internal interpreter
change over the in-repo `internal/cap` + `internal/interp` packages.

### Current state (verified by reading source)

- `internal/interp/interp.go::emitStdout` is the SINGLE capability gate every
  stdout effect flows through. Its `!ip.caps.Has(cap.Stdout)` branch currently
  returns `nil` and performs no write — a silent skip (the US-023 placeholder,
  explicitly documented to be replaced by US-024).
- The only caller is `internal/interp/host.go::evalHostCall`, intercepting
  `fmt.Println`. That call site holds the source position (`sel.Pos()`), which
  is what makes the refusal LOCATABLE.
- `internal/cap` exposes `Capability` (with `String()`), `CapabilitySet.Has`,
  `GrantAll`, `DenyAll`. It is dependency-free, so reusing it does not breach
  the US-022 no-go/types / no-typecheck dependency gate.

### Approach (chosen)

Define a named, located capability-denied error in `internal/interp` carrying
the denied `cap.Capability` and the source `token.Pos`; thread the position
into `emitStdout`, and on a denied capability return that error WITHOUT calling
`write` (nothing is written). This is the minimal, in-pattern change — it mirrors
the interpreter's other loud, located refusals (gate(), unresolved-host-symbol).

### Alternatives considered

- Panic via `panicSignal` instead of a returned error: rejected — a capability
  denial is a host-policy refusal of an effect, not a program panic; returning a
  typed error lets a host/test match it with `errors.As` (cleaner than string
  matching on a panic value).
- Generic "effect denied" string error with no capability/position: rejected —
  AC requires the error be NAMED (identifies the capability) and LOCATED.

## Confidence

High — the seam, the single call site, and the cap API are all read directly
from the tree; the change is local and pattern-matching existing refusals.

## Open Questions

None blocking. Only Stdout is routed today; the typed error is written to
generalize to Time/Env/File/Net when those effect sites land.
