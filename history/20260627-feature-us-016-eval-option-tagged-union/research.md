# Research — US-016

This is an internal codebase change with a directly analogous precedent (US-015,
Result as tagged union). No external/web research required.

## Findings

- `internal/interp/value.go` already defines the Result constants + `VariantVal`
  (single ctor for enum/Result/Option) + `payloadValue` (returns ok=false for a
  data-less variant — exactly right for `Option.None`). The note at value.go:189
  explicitly anticipates Option.Some reusing `payloadValue`.
- `internal/interp/eval.go` `evalCallMulti` intercepts `Result.Ok/Err` via
  `evalResultCtor` (receiver name guard + not-shadowed). `Option.Some(x)` is the
  same node shape (positional `*ast.CallExpr` with `*ast.SelectorExpr` Fun) and
  is intercepted the same way.
- `Option.None` (no parens) is a `*ast.SelectorExpr`. `evalSelector` already
  intercepts data-less enum construction via `enumByName`, but `Option` is NOT in
  `info.Enums`, so it needs its own small guard returning
  `VariantVal("Option","None",nil)`.
- `interp.go` `armScopeFor` unwraps the payload only for `TypeID == resultTypeID`;
  extend the guard to also fire for `optionTypeID`. `payloadValue` handles None
  (zero fields -> ok=false -> binds nothing) for free.
- `selectMatchArm` keys on `Variant.Tag` only — Some/None dispatch with no change.
- Corpus shape (04-option, features/04-option/examples/option_exists.goal):
  `match find(id) { Option.Some(u) => ...; Option.None => ... }`.

## Confidence

High — direct, mechanical mirror of the shipped US-015 Result implementation,
confirmed against the existing source and the 04-option corpus fixtures.
