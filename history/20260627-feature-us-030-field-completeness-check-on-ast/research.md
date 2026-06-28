# Research — US-030 field-completeness check on AST

Codebase research (no external sources needed — this mirrors an existing
in-repo check onto the new AST front-end).

## Reference implementation (to mirror for message parity)

`internal/check/fields.go` — the legacy token-scanning field-completeness check.
Key behaviors and exact diagnostic strings:

- Struct literal omission (Error, code `missing-field`):
  `struct literal `T{…}` omits required field(s) `a`[, `b`] — set it/them
  explicitly, or add `...defaults` to fill the rest with zero values`
- Variant construction omission (Error, code `missing-field`):
  `variant construction `E.V(…)` omits required field(s) `a` — a variant has no
  `...defaults`; name every field`
- Unresolved literal type (Warning, code `unresolved-literal-type`):
  `cannot verify field completeness of `T{…}`: type `T` is not declared in this
  file — field-completeness deferred`
- `...defaults` (§8.5) and `...derive(src)` (§12) spreads ⇒ complete by
  construction ⇒ no diagnostic.
- A `Enum.Variant(a)` payload binding inside a match arm is skipped (legacy uses
  brace-span heuristics; AST uses node-type distinction).

## AST facts that make this correct by construction

- `*ast.CompositeLit{Type, Elts}` — struct literal when `Type` is `*ast.Ident`.
  Keyed elements are `*ast.KeyValueExpr` (Key is an `*ast.Ident`). A
  `...defaults`/`...derive(s)` element is `*ast.SpreadElement`.
- `*ast.VariantLit{Enum, Variant, Args}` — variant construction; labeled args are
  `*ast.LabeledArg{Label}`. Built by the parser ONLY when a call carries ≥1
  labeled arg, so a data-less `Enum.Dot` stays a `*ast.SelectorExpr` (trivially
  complete; never a VariantLit).
- `*ast.VariantPattern` (match arm binding) is a DISTINCT node from VariantLit, so
  walking VariantLit alone never sees a pattern — the legacy false positive
  (match_binding_arm.goal) is structurally impossible.
- `sema.Info` already resolves `Structs` (ordered `[]Field`) and `Enums`
  (`Variants` with ordered `Fields`, plus `FieldSet`), populated by Resolve.
  resolveTypeDecl reads struct field types off the AST, so a struct TAG never
  shifts field-name parsing (tagged_struct.goal is clean by construction).

## Existing seams reused

- `sema.Check` aggregator (check.go) — append `CheckFields` alongside
  `CheckExhaustive`.
- `corpus.SemaCheck` already converts every `sema.Diagnostic` to a
  `check.Diagnostic`, so the new check flows to the corpus runner with no adapter
  change.
- `plural`/`pronoun` helpers already in sema/check.go; add a `quoteJoin`.

## Confidence

High — this is a structural port of a passing in-repo check onto already-resolved
AST facts, validated against 9 concrete golden cases with inline `// want`
markers.
