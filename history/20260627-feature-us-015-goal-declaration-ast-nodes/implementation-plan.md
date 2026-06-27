# Implementation Plan — US-015 Goal Declaration AST Nodes

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/ast/goal_decl.go` | The goal-specific declaration nodes: `FuncMod` enum, `EnumDecl`, `Variant`, `PayloadField`, `SealedInterfaceDecl`, `ImplementsClause`, each with `Pos()`/`End()` and (for decls) the `declNode()` marker. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/ast/ast.go` | Add `Mod FuncMod` + `ModPos token.Pos` fields to `FuncDecl`; update `FuncDecl.Pos()` to return `ModPos` when modified. Add optional `Implements *ImplementsClause` field to `StructType`. |
| `internal/ast/walk.go` | Add Walk switch cases for `EnumDecl`, `Variant`, `PayloadField`, `SealedInterfaceDecl`, `ImplementsClause`; extend the `*StructType` case to descend into `Implements`. |
| `internal/ast/ast_test.go` | Add `TestWalkGoalDeclChildren` asserting Walk descends into each new node's children, plus the from/derive modifier/position behavior. |

## Package Structure

```
internal/ast/
  ast.go         (existing — Go-subset nodes; FuncDecl + StructType edited)
  walk.go        (existing — Visitor + Walk; new cases added)
  goal_decl.go   (NEW — goal declaration nodes)
  ast_test.go    (existing — new test added)
```

## Dependency Graph

1. `internal/token` (exists) — Pos/Kind.
2. `internal/ast/ast.go` core nodes (exist) — Node/Decl interfaces, FuncDecl, StructType.
3. `internal/ast/goal_decl.go` (new) — depends on 1 and 2 (Expr, Ident, FieldList, declNode marker).
4. `internal/ast/walk.go` cases (edit) — depends on 3.
5. `internal/ast/ast_test.go` (edit) — depends on 3 and 4.

## Interface Contracts

```go
type FuncMod int
const (
    FuncPlain FuncMod = iota
    FuncFrom
    FuncDerive
)

// FuncDecl gains:
//   Mod    FuncMod
//   ModPos token.Pos
// FuncDecl.Pos() returns ModPos when Mod != FuncPlain && ModPos != token.Pos{}.

type EnumDecl struct {
    Enum     token.Pos
    Name     *Ident
    Lbrace   token.Pos
    Variants []*Variant
    Rbrace   token.Pos
}
func (*EnumDecl) declNode() {}

type Variant struct {
    Name    *Ident
    Lbrace  token.Pos
    Payload []*PayloadField // nil for a data-less variant
    Rbrace  token.Pos
}

type PayloadField struct {
    Name *Ident
    Type Expr
}

type SealedInterfaceDecl struct {
    Sealed    token.Pos
    Interface token.Pos
    Name      *Ident
    Methods   *FieldList
}
func (*SealedInterfaceDecl) declNode() {}

type ImplementsClause struct {
    Implements token.Pos
    Type       Expr
}

// StructType gains: Implements *ImplementsClause
```

All five `Pos()`/`End()` follow the existing convention (first-token Pos,
just-past-last-token End, with nil-child fallbacks like `Field`/`FieldList`).

## Integration Points

- `EnumDecl` and `SealedInterfaceDecl` implement `Decl` (via `declNode()`), so
  they slot into `ast.File.Decls`.
- `Variant`, `PayloadField`, `ImplementsClause` are support nodes implementing
  only `Node` (mirroring `Field`/`FieldList`).
- `ImplementsClause` reaches the tree via `StructType.Implements`; `Walk`'s
  existing `*StructType` case gains a descent into it.
- New `Walk` cases reuse `walkExpr` for `Expr` children and direct `Walk` calls
  for `*Ident`/`*FieldList`/slice children, exactly as the existing cases do.

## Testing Strategy

Add `TestWalkGoalDeclChildren` to `internal/ast/ast_test.go`, reusing the
existing `collector` Visitor. For each new node, construct it with children,
`Walk` it, and assert via a `visits` map that the parent is visited once and
each declared child is visited once:
- EnumDecl → Name + each Variant; Variant(payload) → Name + PayloadField;
  PayloadField → Name + Type; Variant(data-less) → Name only.
- SealedInterfaceDecl → Name + Methods (+ a Field within).
- StructType{Implements} → ImplementsClause + Fields; ImplementsClause → Type.
- FuncDecl{Mod: FuncFrom/FuncDerive} → walks normally (Name visited); assert
  `Mod` is recorded and `Pos()` returns `ModPos`.

All assertions use stdlib `testing` only (no testify), matching the project.
Gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
