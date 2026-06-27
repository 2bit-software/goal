# Technical Requirements / Research — US-030

## Approach

Add `CheckFields(file *ast.File, info *Info) []Diagnostic` to internal/sema and
wire it into `Check`. It walks the AST collecting:

- `*ast.CompositeLit` whose `Type` is an `*ast.Ident` → struct literal. If the
  literal contains a `*ast.SpreadElement` (`...defaults` / `...derive(…)`) it is
  complete by construction → skip. If the Ident names a known `info.Structs`
  entry, compare present keyed fields (`*ast.KeyValueExpr` keys) against the
  declared field order → Error on omissions. If the Ident is unknown but the
  literal is keyed → Warning ("field-completeness deferred").
- `*ast.VariantLit` → enum variant construction. Resolve the enum via
  `exprName(vl.Enum)`; skip unknown enums silently (mirrors legacy). A data-less
  variant is trivially complete. Otherwise compare present `*ast.LabeledArg`
  labels against the declared variant field order → Error on omissions.

## Why the AST makes this correct by construction

The legacy check (internal/check/fields.go) reconstructs literal vs block vs
match-pattern shape from a flat token stream with brace-span heuristics. On the
AST, a match-arm payload binding is a `*ast.VariantPattern` and a construction is
a `*ast.VariantLit` — different node types — so the match-binding false positive
(testdata/check/08-no-zero-value/match_binding_arm.goal) is impossible. Struct
tags (tagged_struct.goal) are carried on the AST field, never confused for the
type. Array/map/slice literals have non-Ident `Type`, so they are never mistaken
for struct literals.

## Diagnostic message parity

Messages mirror internal/check/fields.go so the inline `// want` markers match:

- struct omission: `struct literal `T{…}` omits required field(s) `a`[, `b`] — set
  it/them explicitly, or add `...defaults` to fill the rest with zero values`
- variant omission: `variant construction `E.V(…)` omits required field(s) `a` —
  a variant has no `...defaults`; name every field`
- deferred: `cannot verify field completeness of `T{…}`: type `T` is not declared
  in this file — field-completeness deferred`

## Test

Add a corpus runner test mirroring TestSemaExhaustiveRunner that drives every
testdata/check/08-no-zero-value/ case through SemaCheck via RunCheck.

## Constraints

- Zero-dependency, stdlib `testing` only (no testify).
- Reuse `plural`/`pronoun` helpers already in sema/check.go; add `quoteJoin`.
