# Technical Requirements / Research — US-014

## Source of truth
- REWRITE-ARCHITECTURE.md §3: `ast` package = "node types + Visitor/Walk;
  declarations, statements, expressions, patterns" — the one structural model.
- internal/token already provides Kind and Pos{Offset,Line,Col} and Token.

## Design hints (modeled on go/ast, trimmed to the Go subset goal uses)
- Node interface with `Pos() token.Pos` (and `End()` optional but useful).
- Marker interfaces: Decl, Stmt, Expr (each embeds Node) so the parser and
  backends can switch on category.
- File: package name + imports + top-level decls.
- Decls: GenDecl (import/const/var/type via a token.Kind tok + specs),
  FuncDecl. Specs: ImportSpec, ValueSpec, TypeSpec.
- Exprs: Ident, BasicLit, BinaryExpr, UnaryExpr, ParenExpr, SelectorExpr,
  IndexExpr, SliceExpr, CallExpr, StarExpr, CompositeLit, KeyValueExpr,
  FuncLit, plus type-exprs ArrayType, MapType, StructType, InterfaceType,
  FuncType, ChanType, Ellipsis. FieldList/Field for params/results/struct fields.
- Stmts: BlockStmt, ExprStmt, AssignStmt, ReturnStmt, IfStmt, ForStmt,
  RangeStmt, SwitchStmt, CaseClause, DeferStmt, GoStmt, BranchStmt, DeclStmt,
  IncDecStmt, SendStmt, LabeledStmt, EmptyStmt.
- Walk(v Visitor, node Node): pre-order; v.Visit(node) returns a Visitor; if
  non-nil, recurse into children, then call v.Visit(nil) (go/ast convention).
  The acceptance test counts a node once via the pre-order v.Visit(node!=nil).

## Test plan
- Build a File tree by hand covering a representative spread of node types and a
  Visitor that counts nodes; assert the count equals the number of nodes built
  (each visited exactly once).
