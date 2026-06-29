# Technical requirements / research — SEAM-CAP-2

## Root cause (verified)

- `internal/sema/foreign.go` `foreignDecls(dir, alias)` reads ONLY `.go` files (and
  reconstructs foreign enums from their generated §8.1 marker-interface encoding via
  `reconstructForeignEnums`). The real per-package `goal build` resolves an imported
  sibling package to its `.goal` SOURCE dir (via `moduleResolve`), which has no `.go`,
  so `foreignDecls` returns empty and the importer never sees the enum.
- Backend bare construction lowering (`emit.go` `selectorExpr`, `variantLit`,
  `armBodyType`) only recognizes an enum whose base is a bare `*ast.Ident`, so a
  package-qualified `pkg.Enum.Variant` (base is `*ast.SelectorExpr`) lowers verbatim.
  Match lowering already handles the qualified case (SEAM-CAP: `matchQualifier` returns
  `pkg.Enum`); it only needs the enum present in `info.Enums["pkg.Enum"]`.

## Approach (bounded)

1. `foreignDecls`: when the resolved dir has no non-test `.go` files but has `.goal`
   files, parse those `.goal` files with the goal front-end (`parser.ParseFile` +
   `ResolvePackage`) and project the package's EXPORTED enums into `info.Enums` keyed
   `alias.Enum` (mirroring the `.go` reconstruction shape: Variants/VSet/FieldSet).
   Structs/funcs/methods from `.goal` source are out of scope here (enum keystone only;
   strictly additive — today a `.goal`-only dir yields nothing).
2. Backend: add an `enumRef` helper that yields the enum key for both a bare `*ast.Ident`
   and a package-qualified `pkg.Enum` `*ast.SelectorExpr`; use it in `selectorExpr`,
   `variantLit`, and `armBodyType` so bare cross-package construction lowers to
   `pkg.Enum(pkg.Enum_Variant{})`.
3. Mirror BOTH changes in `selfhost/sema/foreign.goal` and `selfhost/backend/*.goal` so
   the self-host stays consistent and `task fixpoint` holds.

## Proof

- New fixture: two sibling `.goal` packages under `internal/backend/testdata/` — a
  defining package with `enum` + a consumer with cross-package `match` and bare
  construction. A test transpiles BOTH packages per-package (real topology), asserts the
  consumer's Go contains the §8.1 type-switch + construction form, and builds+runs them
  together against a reference switch.

## Gates

`task check`, `task build`, `task fixpoint` (both bootstrap stages byte-identical),
corpus behavioral tier unchanged.
