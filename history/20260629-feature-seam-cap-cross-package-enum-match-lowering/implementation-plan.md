# Implementation Plan

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/backend/testdata/extenum/extenum.go` | Foreign Go fixture carrying the generated §8.1 encoding of a tag-only enum `Light { On Off }` (marker interface + variant structs + marker methods), imported by the cross-package fixtures. |
| `testdata/package/cross-pkg-enum/use.goal` | Corpus ModePackage fixture: a goal package that `match`es over the imported enum `light.Light` in statement and return position. |
| `internal/backend/crosspkg_enum_test.go` | Behavioral unit test: transpile the cross-package package, assert no error, and assert the lowered switch maps each variant the same as the equivalent hand-written Go type-switch. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/backend/lower.go` | `matchQualifier`: return the qualified `pkg.Enum` string when the first arm's `vp.Enum` is a `*ast.SelectorExpr` (currently only `*ast.Ident`). |
| `internal/sema/foreign.go` | `foreignDecls` returns a 4th map of reconstructed foreign enums keyed `alias.Enum`; `EnrichForeign` folds them into `info.Enums`. Reconstruction recognizes the §8.1 encoding (marker interface `is`+Name + variant structs `Name_Variant`). |
| `selfhost/backend/lower.goal` | Mirror of the lower.go `matchQualifier` change. |
| `selfhost/sema/foreign.goal` | Mirror of the foreign.go reconstruction change. |
| `corpus/manifest.json` | Register the new ModePackage case with its `imports` map wiring the foreign fixture. |

## Dependency Graph

1. `internal/sema/foreign.go` foreign-enum reconstruction (no deps).
2. `internal/backend/lower.go` `matchQualifier` SelectorExpr handling (no deps).
3. Foreign fixture `internal/backend/testdata/extenum/extenum.go` (no deps).
4. Corpus goal fixture + manifest entry (depends on 1-3 for transpile to succeed).
5. Behavioral unit test (depends on 1-3).
6. selfhost mirrors of 1 and 2 (independent; keep fixpoint green + capability in self-host).

## Interface Contracts

```go
// lower.go
func matchQualifier(m *ast.MatchExpr) string // *ast.Ident -> "Enum"; *ast.SelectorExpr pkg.Enum -> "pkg.Enum"

// foreign.go
func foreignDecls(dir, requestedAlias string) (
    structs map[string][]Field, funcs, methods map[string]FuncSig,
    enums map[string]*Enum, err error) // enums keyed "alias.Enum"
// EnrichForeign also: if info.Enums == nil { info.Enums = map[string]*Enum{} }
//   for name, en := range enums { info.Enums[name] = en }
```

The case-label builder in `enumMatch` already emits `enumName + "_" + Variant`,
so a qualifier of `"light.Light"` yields `case light.Light_On:` — the correct
reference to the imported variant struct. No emit.go change needed.

## Integration Points

- `backend.TranspilePackage` (package.go) -> `sema.ResolvePackage` then
  `enrichForeign` -> `sema.EnrichForeign`: the merged `info.Enums` now contains
  imported enums, so `enumOf(e.info, "light.Light")` resolves and `matchStmt` /
  return-position match dispatch to `enumMatch`.
- Corpus `RunPackage` resolves imports via `DefaultResolver` (module-relative)
  at transpile time and wires the `imports` map into the temp build module.

## Testing Strategy

- Behavioral unit test in `internal/backend`: build a `project.Package` for the
  goal fixture dir, call `TranspilePackage`, assert no error and that the
  generated Go contains `case light.Light_On:` / `case light.Light_Off:`; run a
  table of subject->expected outputs against a hand-written reference switch to
  assert identical mapping.
- Corpus ModePackage case `cross-pkg-enum`: the package runner asserts the
  generated Go is valid and `go build`s with the foreign import wired in — the
  cross-package link proof.
- Gates: `task check`, `task build`, `task fixpoint`; corpus behavioral tier
  unchanged (new case is additive).
