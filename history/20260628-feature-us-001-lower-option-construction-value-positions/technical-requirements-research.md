# Technical Requirements / Research

## Mechanism (from prd.json notes)

Same mechanism as the already-landed nested-Option fix (`optionValueExpr`):
None -> nil, Some(addressable) -> &x, Some(other) -> boxed temp. Generalize it into
the value-emission path (`emitter.expr`). Today only `emitOptionReturn` and the
`Result.Ok` payload call `optionValueExpr`.

## Plan

- internal/backend/emit.go `expr()` is the single value-emission seam: var/const
  values (`spec` ValueSpec), `:=` RHS (`stmt` AssignStmt), call args, composite
  literal fields/elements all route through it. Intercept Option construction at
  the top of `expr()`.
- A non-addressable `Some(x)` in a pure-expression position cannot hoist a temp
  statement, so box it through a generic helper `func goalSome[T any](v T) *T { return &v }`,
  injected once per file (single-file `file()`) / once per package
  (`TranspilePackage`), mirroring the `resultPrelude` / fmt-import injection.
- Classify with a shared `optionConstruction(x)` helper (lower.go) so the emit
  branch and the `needsOptionPrelude` static scan never drift.

## Tests

Backend test (stdlib `testing`, NO testify — project is zero-dependency)
exercising Option construction in var-assignment, call-argument, struct-field, and
slice/map-literal positions; assert valid Go (go/format), no `Option.` token, and
the `nil` / `&x` / boxed encodings.
