# Implementation Plan — US-014 Core AST nodes and Walk

## Files
- `internal/ast/ast.go` — Node/Decl/Stmt/Expr interfaces + all Go-subset node
  structs, each with `Pos()` (and `End()` where cheap). Imports only
  `goal/internal/token`.
- `internal/ast/walk.go` — `Visitor` interface, `Walk(v Visitor, node Node)`,
  and a `walkList` helper. Pre-order, go/ast convention.
- `internal/ast/ast_test.go` — `package ast`, stdlib `testing` only. Builds a
  hand-made `*File` tree and a counting Visitor; asserts Walk visits every node
  exactly once.

## Node set (Go subset goal uses; modeled on go/ast)
- Top: `File{Package token.Pos, Name *Ident, Imports []*ImportSpec, Decls []Decl}`.
- Support: `Field{Names []*Ident, Type Expr, Tag *BasicLit}`,
  `FieldList{Opening token.Pos, List []*Field, Closing token.Pos}`.
- Decls: `GenDecl{TokPos, Tok token.Kind, Specs []Spec}`, `FuncDecl{Recv *FieldList,
  Name *Ident, Type *FuncType, Body *BlockStmt}`. Marker `Decl`.
- Specs (`Spec` marker): `ImportSpec{Name *Ident, Path *BasicLit}`,
  `ValueSpec{Names []*Ident, Type Expr, Values []Expr}`,
  `TypeSpec{Name *Ident, Type Expr}`.
- Exprs (`Expr` marker): `Ident`, `BasicLit`, `ParenExpr`, `UnaryExpr`,
  `BinaryExpr`, `SelectorExpr`, `IndexExpr`, `SliceExpr`, `CallExpr`, `StarExpr`,
  `CompositeLit`, `KeyValueExpr`, `FuncLit`, and type-exprs `ArrayType`,
  `MapType`, `StructType`, `InterfaceType`, `FuncType`, `ChanType`, `Ellipsis`.
- Stmts (`Stmt` marker): `BlockStmt`, `ExprStmt`, `AssignStmt`, `ReturnStmt`,
  `IfStmt`, `ForStmt`, `RangeStmt`, `SwitchStmt`, `CaseClause`, `DeferStmt`,
  `GoStmt`, `BranchStmt`, `DeclStmt`, `IncDecStmt`, `EmptyStmt`.

## Interfaces
- `Node interface { Pos() token.Pos; End() token.Pos }`.
- `Decl interface { Node; declNode() }`, `Stmt interface { Node; stmtNode() }`,
  `Expr interface { Node; exprNode() }`, `Spec interface { Node; specNode() }`.
  Private marker methods keep category membership closed to this package.
- `Visitor interface { Visit(node Node) (w Visitor) }`.
- `Walk(v Visitor, node Node)`: `if v = v.Visit(node); v == nil { return }`,
  then a type switch recursing into each node's children (using `walkList` for
  slices and nil-guards for optional fields), then `v.Visit(nil)`.

## Walk contract / nil handling
- `Walk(v, nil)` and nil child fields are no-ops (guarded before recursion).
- A node is counted once via the non-nil `v.Visit(node)` call; `v.Visit(nil)`
  end-markers are not counted by the test's counting visitor (it only counts
  non-nil arguments).

## Dependency order
1. `ast.go` (types) → compiles against existing `token`.
2. `walk.go` (Walk) → depends on the node types.
3. `ast_test.go` → depends on both.

## Test strategy (acceptance)
Build a `*File` containing a representative spread (package, import spec, a
GenDecl/ValueSpec, a FuncDecl with params/results FieldList, a BlockStmt with
Assign/If/For/Return/Expr/Defer statements, and nested expressions incl.
Binary/Unary/Selector/Call/Composite/Star/Index). Maintain a hand-counted total
of constructed nodes; a counting Visitor increments on each non-nil Visit;
assert `count == expectedTotal`. Also assert `Walk(v, nil)` does not panic.

## Out of scope
Goal-specific nodes (US-015/016), parser (US-017+), printing (US-025+).
