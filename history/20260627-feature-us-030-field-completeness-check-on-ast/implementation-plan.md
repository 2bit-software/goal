# Implementation Plan — US-030 field-completeness check on AST

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/sema/fields.go` | `CheckFields(file, info)` — the §8 field-completeness check over the AST (struct literals + variant constructions), plus a `quoteJoin` helper. |
| `internal/corpus/sema_fields_test.go` | Corpus runner test driving every `testdata/check/08-no-zero-value/` case through `SemaCheck` via `RunCheck`. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/sema/check.go` | `Check` appends `CheckFields(file, info)...` after `CheckExhaustive`. |

## Package Structure

```
internal/sema/
  check.go      (modified: wire CheckFields into Check)
  fields.go     (new: CheckFields + helpers)
internal/corpus/
  sema_fields_test.go (new: corpus runner test)
```

## Dependency Graph

1. `internal/sema/fields.go` — depends only on existing ast/token/sema.Info.
2. `internal/sema/check.go` — wires (1) into the aggregator.
3. `internal/corpus/sema_fields_test.go` — drives (2) through the corpus runner.

## Interface Contracts

```go
// internal/sema/fields.go
func CheckFields(file *ast.File, info *Info) []Diagnostic

// internal/sema/check.go
func Check(file *ast.File, info *Info) []Diagnostic // += CheckFields(...)
```

Diagnostic emission (severities/codes/messages mirror internal/check/fields.go):
- struct omission: Error, code `missing-field`,
  ``struct literal `T{…}` omits required field(s) `a`[, `b`] — set it/them explicitly, or add `...defaults` to fill the rest with zero values``
- variant omission: Error, code `missing-field`,
  ``variant construction `E.V(…)` omits required field(s) `a` — a variant has no `...defaults`; name every field``
- unresolved: Warning, code `unresolved-literal-type`,
  ``cannot verify field completeness of `T{…}`: type `T` is not declared in this file — field-completeness deferred``

Feature string for all three: `08-no-zero-value`.

## Algorithm

Walk the file (reuse `collectMatches`'s `visitorFunc` pattern):

- For each `*ast.CompositeLit` whose `Type` is `*ast.Ident name`:
  - If any elt is `*ast.SpreadElement` → complete by construction; skip.
  - present = { key.Name | elt is `*ast.KeyValueExpr` with `*ast.Ident` key }.
  - If `info.Structs[name]` known: missing = declared fields not in present (in
    order) → if non-empty, Error.
  - Else if the literal is keyed (≥1 KeyValueExpr): Warning (deferred).
  - Else skip.
- For each `*ast.VariantLit`:
  - enumName = exprName(vl.Enum); enum = info.Enums[enumName]; if nil → skip.
  - variant = enum's variant matching vl.Variant.Name; if absent → skip; if
    data-less → clean.
  - present = { arg.Label.Name | arg is `*ast.LabeledArg` }.
  - missing = declared variant fields not in present (in order) → Error.

`exprName` already exists in sema/check.go (reused). `plural`/`pronoun` reused;
add `quoteJoin(names []string) string` (backtick-join, no enum prefix).

## Integration Points

- `internal/sema/check.go` `Check` is the single aggregator the corpus
  `SemaCheck` adapter calls — appending here is the only wiring needed; the
  adapter already forwards all sema diagnostics to `check.Diagnostic`.

## Testing Strategy

`internal/corpus/sema_fields_test.go`: mirror `sema_checker_test.go`
(`TestSemaExhaustiveRunner`) — iterate manifest `KindCheck` cases whose Input has
prefix `testdata/check/08-no-zero-value/`, run each through
`RunCheck(repoRoot, c, CheckerFunc(SemaCheck))`, `t.Fatalf` if zero cases ran.
The 9 golden cases (with inline `// want` markers and no-marker clean cases) are
the assertions. Plus the prd verifyCommands (build, vet, full test).
