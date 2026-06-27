# Technical Requirements / Research — US-012

## Existing seams (already in place)

- `internal/interp/value.go` already defines the universal tagged union
  `Variant{TypeID, Tag, Fields}` and `VariantVal(typeID, tag, fields)`, plus
  `Value.Field(name)` to read a payload field. No value-model change is needed.
- `internal/sema.Info.Enums` maps an enum name to `*sema.Enum{Variants, VSet,
  FieldSet}` — the native fact source for validating the enum + variant exist.

## Parser shapes (from existing parser, confirmed)

- A payload construction with labeled args (`Status.Active(since: now())`) parses
  to `*ast.VariantLit{Enum, Variant, Args:[]*ast.LabeledArg}`. `Enum` is the
  enum type reference (`*ast.Ident`), `Variant` is the tag (`*ast.Ident`).
- A data-less construction (`Status.Pending`) parses to `*ast.SelectorExpr`
  (no parens). It must be intercepted in `evalSelector`, guarded by enum
  membership in `info.Enums` + the tag in the enum's `VSet`, and only when the
  receiver ident is not shadowed by a value binding in scope.

## Plan

1. `evalExpr`: add `case *ast.VariantLit` → `evalVariantLit`.
2. `evalVariantLit`: resolve the enum name (the `*ast.Ident` Enum ref) and tag,
   validate against `info.Enums`/VSet, evaluate each `*ast.LabeledArg` into the
   Fields map (also map a positional arg by declared field order, for
   robustness), and return `VariantVal(enum, tag, fields)`.
3. `evalSelector`: before evaluating the receiver, intercept the data-less enum
   variant case and return `VariantVal(enum, tag, nil)`.

## Constraints

- Zero-dependency: stdlib `testing` only, no testify.
- `internal/interp` must not gain a dependency on `internal/backend`,
  `internal/typecheck`, or `go/types` (US-022 gate). Reading `sema.Info` is fine.
- Loud, descriptive refusals — never a silent nil.
