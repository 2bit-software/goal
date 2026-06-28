# Technical Requirements / Research — US-016

Mirror the US-015 Result encoding exactly (progress.txt US-015 learnings):

- Add Option constants to `internal/interp/value.go`: `optionTypeID = "Option"`,
  `optionSomeTag = "Some"`, `optionNoneTag = "None"`, `optionSomeField = "value"`.
- `Option.Some(x)` is a bare positional call -> `*ast.CallExpr` with a
  `*ast.SelectorExpr` Fun. Intercept it in `evalCallMulti` by receiver name
  `Option` (guarded by "not shadowed"), the same way `Result` is intercepted ->
  new `evalOptionCtor` building `VariantVal("Option","Some",{value:x})`.
- `Option.None` has no parens -> `*ast.SelectorExpr`. Intercept it in
  `evalSelector` (Option is NOT in `info.Enums`, so it needs its own guard,
  not shared with the enum data-less path) -> `VariantVal("Option","None",nil)`.
- Extend `armScopeFor` in `interp.go` so the unwrap guard fires for
  `TypeID == optionTypeID` too. `payloadValue` already returns `ok=false` for
  None's zero fields, so a None arm binds nothing — correct.
- Match dispatch (`selectMatchArm`) keys on `Variant.Tag` only, so Some/None
  dispatch for free — no match change needed.
