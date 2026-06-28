# Implementation Plan — US-020 Eval defaults expansion

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/defaults_test.go` | Unit tests over an 08-no-zero-value/defaults shape proving omitted fields receive zeros, set fields are preserved, nested struct/slice defaults, and a `...derive` spread is refused. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/interp/eval.go` | In `evalCompositeLit`'s struct (`*ast.Ident`) case, accept a `*ast.SpreadElement` element; for `...defaults`, fill omitted declared fields with their runtime zero. Add a local `zeroValue(typ string) Value` helper and a `baseType` string helper (or inline) for type-name normalization. |

## Package Structure

No new packages. All work is inside the existing `internal/interp` package.

```
internal/interp/
  eval.go            (modified: evalCompositeLit + zeroValue helper)
  defaults_test.go   (new)
```

## Dependency Graph

1. `zeroValue(typ string) Value` helper in eval.go — depends only on value.go
   constructors and `ip.info.Structs`.
2. `evalCompositeLit` struct-case change — depends on (1).
3. `defaults_test.go` — depends on (2).

## Interface Contracts

```go
// zeroValue returns the runtime zero Value for a declared sema type string,
// mirroring backend zeroLit: ""/false/0 for primitives, NilVal for
// ptr/map/chan/func/non-empty-iface/any/error, an empty SliceVal for []T, and a
// recursively zero-filled StructVal for a named in-file struct. depth guards
// recursive struct nesting.
func (ip *Interp) zeroValue(typ string, depth int) Value
```

Struct case of `evalCompositeLit` becomes:
- Iterate `c.Elts`. A `*ast.KeyValueExpr` sets `fields[name]` as today.
- A `*ast.SpreadElement` whose `X` is `*ast.Ident{Name:"defaults"}` records a
  `wantDefaults` flag; any other spread is a descriptive, located refusal
  (`interp: unsupported spread element ... in composite literal`).
- A non-KeyValue, non-Spread element keeps the existing "requires keyed
  field: value" refusal.
- After the loop, if `wantDefaults`: look up `ip.info.Structs[t.Name]`; if
  unknown, descriptive refusal; else for each declared `Field` not present in
  `fields`, set `fields[f.Name] = ip.zeroValue(f.Type, 0)`.
- Return `StructVal(t.Name, fields)`.

## Integration Points

- `internal/interp/eval.go` `evalCompositeLit` (the `case *ast.Ident:` branch),
  reached from `evalExpr`'s `*ast.CompositeLit` case.
- Reads `ip.info.Structs` (`map[string][]sema.Field`) populated by
  `sema.Resolve` (already wired through `New`).

## Testing Strategy

- New `internal/interp/defaults_test.go`, `package interp`, stdlib `testing`
  only (no testify), modeled on composite_test.go / enum_test.go.
- Drive 08-no-zero-value/defaults-shaped programs through `newInterp` + `evalFn`:
  - primitives: `User{name:..., role:..., ...defaults}` -> email "", active
    false, logins 0; name/role preserved.
  - nested struct + slice defaults -> zeroed Addr{}, empty slice.
  - explicit-before-and-after `...defaults` ordering preserves set fields.
  - hand-built `...derive` spread -> descriptive error (evalFnErr).

## Requirement -> Plan Map

- FR-1, FR-2 -> evalCompositeLit fill loop (omitted-only).
- FR-3 -> zeroValue type table.
- FR-4 -> non-defaults spread refusal.
