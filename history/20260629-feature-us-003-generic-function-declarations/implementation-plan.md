# Implementation Plan

## File Inventory

### New Files

None required (a test is added to an existing backend/parser test file or a new focused test file).

### Modified Files

| File | Changes |
|------|---------|
| `internal/ast/ast.go` | Add `TypeParams *FieldList` to `FuncType` struct. |
| `internal/ast/walk.go` | Walk `n.TypeParams` in the `*FuncType` case. |
| `internal/parser/parser.go` | In `parseFuncDecl`, after `fd.Name`, parse type params via existing `atTypeParams`/`parseTypeParams` when no receiver. |
| `internal/backend/emit.go` | In `funcSig` (or `funcDecl`), emit `t.TypeParams` via `e.fieldList(t.TypeParams, "[", "]")` before the params. |

## Dependency Graph

1. `ast.FuncType.TypeParams` field (foundation).
2. `walk.go` FuncType walk (depends on 1).
3. Parser populates the field (depends on 1).
4. Backend emits the field (depends on 1).
5. Tests (depend on 3, 4).

## Interface Contracts

```go
type FuncType struct {
    Func       token.Pos
    TypeParams *FieldList // generic type params in "[...]"; nil when non-generic
    Params     *FieldList
    Results    *FieldList
}
```

Parser in `parseFuncDecl`:
```go
fd.Name = p.ident()
if fd.Recv == nil && p.atTypeParams() {
    ft.TypeParams = p.parseTypeParams()
}
ft.Params = p.parseParamList()
```

Backend in `funcSig` (emitted between name and params):
```go
if t.TypeParams != nil {
    e.fieldList(t.TypeParams, "[", "]")
}
```

## Integration Points

- Backend: the type-param list must print between the func name and `(` —
  emitted at the start of `funcSig` (which runs right after `d.Name` is printed
  in `funcDecl`).
- Parser reuses the same `atTypeParams`/`parseTypeParams` already used by
  `parseTypeSpec`, so behavior is consistent with generic type declarations.

## Testing Strategy

- A backend transpile test feeding `func Identity[T any](x T) T { return x }`
  and a constrained `[K comparable, V any]` function, asserting the emitted Go
  contains the type-param list and that it round-trips.
- Lean on `selfhost.BuildTranspiled` / existing transpile-and-go-build harness
  if a focused compile assertion is warranted; otherwise assert emitted text.
- Full suite (`task check`) confirms non-generic functions are unchanged.
