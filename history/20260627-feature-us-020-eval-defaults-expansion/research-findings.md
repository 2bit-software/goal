# Research Findings — US-020 Eval defaults expansion

## Summary

`...defaults` is already fully designed elsewhere in the codebase; the
interpreter just needs the runtime analogue of the backend's compile-time
expansion. No external research needed — this is an internal-consistency task.

## How `...defaults` works in the existing front-end

- **Parse**: `...defaults` becomes `*ast.SpreadElement{X: *ast.Ident{Name:
  "defaults"}}` inside a `*ast.CompositeLit.Elts` (internal/ast/goal_expr.go).
- **Static check**: internal/sema `CheckFields` (internal/sema/fields.go) enforces
  feature-08 — a struct literal that omits a field whose zero is *unsafe* (nil
  pointer/map/chan/func/method-iface, or a sum type) is a diagnostic unless that
  field is set explicitly. So by the time a valid program reaches the
  interpreter, every field a `...defaults` would fill has a SAFE zero.
- **Go backend**: internal/backend/lower.go `compositeLit` expands a `...defaults`
  element via `defaultEntries` -> one `name: zeroLit(type)` per omitted declared
  field. `zeroLit` (mirrored from analyze.ZeroLit) maps a sema type string to the
  Go zero literal: ""/false/0 for primitives, nil for ptr/map/chan/func/iface,
  `T{}` for named structs, `nil` for slices.

## Interpreter mapping (the runtime analogue)

The interpreter erases static types but holds `sema.Info.Structs[name] []Field`
(Name + Type string). The runtime zero `Value` is derived from the field's sema
type string, mirroring `zeroLit`:

| sema type | runtime Value |
|-----------|---------------|
| string | StrVal("") |
| bool | BoolVal(false) |
| int/uint/byte/rune kinds | IntVal(0) |
| float32/float64 | FloatVal(0) |
| `*T`, `map[...]`, `chan`, `func`, non-empty `interface`, `any`, `error` | NilVal() |
| `[]T` (slice) | empty SliceVal (usable nil-slice equivalent) |
| named in-file struct | recursively zero-filled StructVal |

## Reference fixtures

- features/08-no-zero-value/examples/defaults_primitives.goal — primitives.
- features/08-no-zero-value/examples/defaults_refs.goal — struct/slice/alias refs.

## Confidence

High — the semantics are pinned by the existing backend lowering and sema check;
the interpreter change is a direct mirror reading sema facts.

## Dependency hygiene

internal/interp must keep its US-022-clean dep set (no internal/backend, go/types,
internal/typecheck). The zero computation is local, reading only sema.Info.
