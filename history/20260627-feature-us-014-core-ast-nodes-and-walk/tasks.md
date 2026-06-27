# Implementation Tasks — US-014 Core AST nodes and Walk

## Task 1: Define AST node types
**Status**: completed
**Files**: internal/ast/ast.go
**Depends on**: (none — uses existing goal/internal/token)
**Spec coverage**: FR-1, FR-2, FR-3, FR-4
**Verify**: `go build ./...`

### Instructions
- New file `internal/ast/ast.go`, package `ast`, import only `goal/internal/token`.
- Define `Node interface { Pos() token.Pos; End() token.Pos }` and the closed
  category markers `Decl`, `Stmt`, `Expr`, `Spec` (each embeds Node + a private
  marker method).
- Define support nodes `Field`, `FieldList`; top node `File`.
- Define decls `GenDecl`, `FuncDecl`; specs `ImportSpec`, `ValueSpec`, `TypeSpec`.
- Define exprs: `Ident`, `BasicLit`, `ParenExpr`, `UnaryExpr`, `BinaryExpr`,
  `SelectorExpr`, `IndexExpr`, `SliceExpr`, `CallExpr`, `StarExpr`,
  `CompositeLit`, `KeyValueExpr`, `FuncLit`, and type exprs `ArrayType`,
  `MapType`, `StructType`, `InterfaceType`, `FuncType`, `ChanType`, `Ellipsis`.
- Define stmts: `BlockStmt`, `ExprStmt`, `AssignStmt`, `ReturnStmt`, `IfStmt`,
  `ForStmt`, `RangeStmt`, `SwitchStmt`, `CaseClause`, `DeferStmt`, `GoStmt`,
  `BranchStmt`, `DeclStmt`, `IncDecStmt`, `EmptyStmt`.
- Each concrete node: a struct carrying child nodes + relevant token.Pos fields,
  with `Pos()`/`End()` methods and its private marker method. Model field shapes
  on go/ast trimmed to goal's subset.

## Task 2: Implement Walk + Visitor
**Status**: completed
**Files**: internal/ast/walk.go
**Depends on**: Task 1
**Spec coverage**: FR-5
**Verify**: `go build ./...`

### Instructions
- New file `internal/ast/walk.go`, package `ast`.
- `type Visitor interface { Visit(node Node) (w Visitor) }`.
- `func Walk(v Visitor, node Node)`: guard nil node; `v = v.Visit(node)`; if nil
  return; type-switch over every concrete node, recursing into children via a
  `walkList`/`walkExprList`/`walkStmtList` helper and nil-guarding optional
  fields; finally `v.Visit(nil)`.
- Pre-order, go/ast convention. Walk(v, nil) is a no-op.

## Task 3: Walk-visits-every-node test
**Status**: completed
**Files**: internal/ast/ast_test.go
**Depends on**: Task 1, Task 2
**Spec coverage**: acceptance criterion (Walk visits each node exactly once)
**Verify**: `go test ./internal/ast/ -count=1`

### Instructions
- New file `internal/ast/ast_test.go`, `package ast`, stdlib `testing` only.
- Build a `*File` by hand covering a representative spread of node types
  (package, ImportSpec, GenDecl/ValueSpec, FuncDecl with params+results
  FieldList, BlockStmt with Assign/If/For/Return/Expr/Defer, and nested
  Binary/Unary/Selector/Call/Composite/Star/Index exprs).
- Track an expected total = the exact number of ast.Node values constructed.
- A counting Visitor increments a counter only when `node != nil`, returns
  itself; run `Walk(counter, file)`; assert counter == expected total (each node
  visited exactly once).
- Add a sub-assertion that `Walk(counter, nil)` does not panic and does not count.

## Final verify (all gates)
- `go build ./...`
- `go vet ./...`
- `go test ./... -count=1`
