# Implementation Tasks — US-015

## Task 1: Add goal declaration nodes + Walk cases + test
**Status**: completed
**Files**: `internal/ast/goal_decl.go` (new), `internal/ast/ast.go`,
`internal/ast/walk.go`, `internal/ast/ast_test.go`
**Depends on**: (none — builds on existing internal/ast)
**Spec coverage**: FR-1..FR-5; all acceptance criteria.
**Verify**: `go build ./...` && `go vet ./...` &&
`go test ./internal/ast/... -count=1` && `go test ./... -count=1`

### Instructions

This story is a single cohesive, independently-committable change to the
`internal/ast` package (4 files, within the 3-5 limit). Build order matches the
plan's dependency graph.

1. **Create `internal/ast/goal_decl.go`** with, following the exact
   Pos()/End()/marker conventions in `ast.go`:
   - `FuncMod int` enum: `FuncPlain` (iota), `FuncFrom`, `FuncDerive`.
   - `EnumDecl{Enum, Name, Lbrace, Variants []*Variant, Rbrace}` —
     `Pos()=Enum`, `End()=Rbrace+1`, `declNode()`.
   - `Variant{Name *Ident, Lbrace, Payload []*PayloadField, Rbrace}` — support
     node (Node only). `Pos()=Name.Pos()`; `End()=Rbrace+1` if braced else
     `Name.End()`.
   - `PayloadField{Name *Ident, Type Expr}` — support node. Pos/End with
     name→type fallbacks (mirror `Field`).
   - `SealedInterfaceDecl{Sealed, Interface, Name, Methods *FieldList}` —
     `Pos()=Sealed`, `End()=Methods.End()` (fallbacks), `declNode()`.
   - `ImplementsClause{Implements token.Pos, Type Expr}` — support node.
     `Pos()=Implements`; `End()=Type.End()` else just past the keyword.

2. **Edit `internal/ast/ast.go`**:
   - `FuncDecl`: add `Mod FuncMod` and `ModPos token.Pos` fields; update
     `FuncDecl.Pos()` to return `ModPos` when `Mod != FuncPlain && ModPos !=
     token.Pos{}` (before the existing Type/Name fallbacks).
   - `StructType`: add `Implements *ImplementsClause` field (Pos/End unchanged —
     Fields still bound the type).

3. **Edit `internal/ast/walk.go`**:
   - Extend the `*StructType` case to `Walk(v, n.Implements)` when non-nil
     (before Fields).
   - Add cases: `*EnumDecl` (Name, then each Variant), `*Variant` (Name, then
     each Payload field), `*PayloadField` (Name, then walkExpr Type),
     `*SealedInterfaceDecl` (Name, then Methods), `*ImplementsClause`
     (walkExpr Type). Place them in a "Goal declarations" group, mirroring the
     nil-guarding style of the existing cases.

4. **Edit `internal/ast/ast_test.go`**: add `TestWalkGoalDeclChildren` reusing
   the existing `collector`. Build each new node with children, `Walk` it, and
   assert via the visits map that the parent and each declared child are each
   visited exactly once. Cover: payload + data-less variant, sealed interface
   with a method Field, struct with ImplementsClause (qualified type), and a
   `from`/`derive` FuncDecl (assert `Mod` recorded and `Pos()==ModPos`).

### Verify
Run all four verify commands; all must be green before committing.
