- **Do** put fallible functions behind `Result[T, error]` and chain with `?`. Don't write
  `v, _ := f()` — it's the exact footgun goal exists to kill.
- **Do** use `Option[T]` for "maybe absent," never a bare `*T` or a sentinel.
- **Do** use `match` (never Go's `switch`) on enums / `Result` / `Option`, and prefer exhaustive
  arms over a reflexive `_` rest-arm — a `switch`'s `default:` silently defeats exhaustiveness.
- **Do** set any field whose zero is unsafe (nil `map`/pointer/`chan`/`func`, method-bearing
  interface, sealed sum) explicitly; safe-zero fields may be omitted. `...defaults` is
  valid-but-redundant now — expect it (and plain omission) to **reject** unsafe-zero fields
  (set those explicitly, or model them with `Option[T]`).
- **Do** add `/// >>> expr` / `/// expected` doctests on pure functions — the cheapest,
  highest-signal feedback band.
- **Do** declare `type T struct implements I` when T is meant to satisfy I — catch drift at the
  type, not at a distant call site.
- **Avoid** identifiers prefixed `__goal_` (reserved for synthesized temporaries).
- **Avoid** unqualified constructors (`Ok(x)`); always `Result.Ok(x)`, `Status.Active(...)`.
- **When unsure what a construct lowers to,** run `go run ./cmd/goalc <file>.goal` and read the
  emitted Go — it is the ground truth.
- **Iterate** with `goal check` first (fastest), then `goal run`, then emit + `go test`.
