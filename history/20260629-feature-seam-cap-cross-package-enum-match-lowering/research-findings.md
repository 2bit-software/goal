# Research findings

## Reproduction (confirmed empirically)

A statement-position `match c { ext.Color.Red => ... ext.Color.Blue => ... }`
over an enum imported from another package fails with the exact error:

    backend: unsupported statement-position match on "" (only Result/Option and enum match are lowered)

Cause confirmed: `matchQualifier` returns `""` because the arm's `vp.Enum` is a
`*ast.SelectorExpr`, not an `*ast.Ident`.

## Key architectural fact

- Single-file `backend.Transpile` resolves with `sema.Resolve` only (no foreign
  enrichment), so cross-package enum resolution is NOT available there.
- Package-mode `backend.TranspilePackage` calls `sema.ResolvePackage` then
  `EnrichForeign`, folding imported packages' facts into one merged `sema.Info`.
  This is the seam where foreign enums must be reconstructed.
- Therefore the regression fixture is a ModePackage corpus case (the established
  pattern, matching `testdata-package-foreign-derive`), whose `imports` map wires
  a foreign Go fixture carrying the generated §8.1 enum encoding.

## Reconstruction shape (from internal/pass genEnum / backend lower.genEnum)

The §8.1 encoding of `enum X { A B }` is:
- `type X interface{ isX() }`           (marker interface, method `is`+X)
- `type X_A struct{}` / `type X_B struct{}`  (variant structs, exported when X is)
- `func (X_A) isX() {}` ...             (marker methods)

So foreign reconstruction: find exported interface `X` whose sole method is
`isX()`, then collect exported struct types named `X_<Variant>` → enum X with
variant set {<Variant>...}. Data-less enums (FuncMod/ChanDir-style) need only
Name + variant set.

## Confidence

High — root cause read directly from source and reproduced with a unit test.
