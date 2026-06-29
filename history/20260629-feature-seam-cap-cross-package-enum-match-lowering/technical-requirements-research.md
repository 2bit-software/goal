# Technical requirements & research

## Live vs mirror

The live transpiler is `internal/` (Go), exercised by `task check` and the
corpus behavioral tier. `selfhost/` is the `.goal` mirror exercised by
`task fixpoint`. Both must be fixed in lockstep (the selfhost port mirrors the
internal Go) so the capability exists in the self-hosted compiler and the
mirror stays valid goal that transpiles self-consistently.

## Root cause (verified)

1. `matchQualifier` (lower.go/lower.goal) returns `""` for a package-qualified
   variant pattern: `vp.Enum` is a `*ast.SelectorExpr` (`pkg.Enum`), but the
   function only handles `*ast.Ident`. With `""` the emitter falls through to
   `unsupported statement-position match on ""`.
2. `enumOf` looks up `info.Enums[name]`, which only ever holds LOCAL-package
   enums (populated by sema.Resolve). Imported enums are absent; only
   Result/Option cross via hardcoded string special-casing.

## Approach

1. `matchQualifier`: when the first arm's `vp.Enum` is a `*ast.SelectorExpr`
   `pkg.Enum`, return the qualified string `"pkg.Enum"`. The case-label
   generation in `enumMatch` already builds `enumName + "_" + Variant`, so
   `"pkg.Enum" + "_" + "Variant"` = `"pkg.Enum_Variant"`, which is exactly the
   correct cross-package reference to the imported §8.1 variant struct.
2. Foreign enum resolution: extend `sema.EnrichForeign` (foreign.go/foreign.goal)
   to reconstruct enums from an imported package's generated §8.1 encoding — a
   marker interface `type X interface{ isX() }` plus variant structs `X_V` with
   a `func (X_V) isX() {}` marker method — and fold them into `info.Enums` keyed
   by the qualified name `alias.X`. This generalizes beyond the Result/Option
   special-casing: any imported user enum is found via the merged (cross-package)
   sema.Info, exactly as foreign structs already are.

## Regression fixture

Add a ModePackage corpus case (established pattern: `testdata/package/...` +
`imports` map wiring a foreign Go fixture). The foreign package carries the
generated §8.1 enum encoding; the goal package `match`es over `pkg.Enum`. The
corpus package runner transpiles, asserts valid Go, and `go build`s — proving it
transpiles and links. Add a focused unit test asserting the lowered switch
behaves identically to the equivalent hand-written switch.

Use a data-less enum (tag-only, like the real FuncMod/ChanDir targets) to keep
reconstruction to Name + variant set (no payload FieldSet needed).

## Gotchas

- Touches the emitter (lower) — watch `task fixpoint`: both bootstrap stages
  must still emit byte-identical Go for the compiler's own source.
- 1Password SSH commit signing fails non-interactively; commit with
  `commit.gpgsign=false`.
