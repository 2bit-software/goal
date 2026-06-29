package ast

// Visitor is invoked by Walk for each node it encounters. If the result Visitor
// w is non-nil, Walk descends into the node's children with w, then calls
// w.Visit(nil) to signal that all children have been visited; if w is nil, Walk
// does not descend into the node.
type Visitor interface {
	Visit(node Node) (w Visitor)
}

// Walk traverses an AST in depth-first pre-order: it visits node, and if
// v.Visit(node) returns a non-nil Visitor w, it recursively walks each of node's
// non-nil children with w and then calls w.Visit(nil). A nil node is a no-op.
//
// Every node in the tree is passed to Visit exactly once (the trailing
// Visit(nil) calls carry a nil node and are not part of that count). New
// goal-specific node types add cases to the switch below.
func Walk(v Visitor, node Node) {
	if node == nil {
		return
	}
	if v = v.Visit(node); v == nil {
		return
	}

	switch n := node.(type) {
	// Support nodes.
	case *Field:
		walkIdentList(v, n.Names)
		walkExpr(v, n.Type)
		if n.Tag != nil {
			Walk(v, n.Tag)
		}
	case *FieldList:
		for _, f := range n.List {
			Walk(v, f)
		}

	// File.
	case *File:
		if n.Name != nil {
			Walk(v, n.Name)
		}
		for _, d := range n.Decls {
			Walk(v, d)
		}

	// Declarations.
	case *GenDecl:
		for _, s := range n.Specs {
			Walk(v, s)
		}
	case *FuncDecl:
		if n.Doc != nil {
			Walk(v, n.Doc)
		}
		if n.Recv != nil {
			Walk(v, n.Recv)
		}
		if n.Name != nil {
			Walk(v, n.Name)
		}
		if n.Type != nil {
			Walk(v, n.Type)
		}
		if n.Body != nil {
			Walk(v, n.Body)
		}
	case *DocComment:
		// no Node children (Lines/Doctests are plain data)

	// Goal declarations.
	case *EnumDecl:
		if n.Name != nil {
			Walk(v, n.Name)
		}
		for _, vr := range n.Variants {
			Walk(v, vr)
		}
	case *Variant:
		if n.Name != nil {
			Walk(v, n.Name)
		}
		for _, f := range n.Payload {
			Walk(v, f)
		}
	case *PayloadField:
		if n.Name != nil {
			Walk(v, n.Name)
		}
		walkExpr(v, n.Type)
	case *SealedInterfaceDecl:
		if n.Name != nil {
			Walk(v, n.Name)
		}
		if n.Methods != nil {
			Walk(v, n.Methods)
		}
	case *ImplementsClause:
		walkExpr(v, n.Type)

	// Goal expressions and patterns.
	case *MatchExpr:
		walkExpr(v, n.Subject)
		for _, arm := range n.Arms {
			Walk(v, arm)
		}
	case *MatchArm:
		walkExpr(v, n.Pattern)
		if n.Body != nil {
			Walk(v, n.Body)
		}
	case *VariantPattern:
		walkExpr(v, n.Enum)
		if n.Variant != nil {
			Walk(v, n.Variant)
		}
		if n.Binding != nil {
			Walk(v, n.Binding)
		}
	case *RestPattern:
		// no children
	case *UnwrapExpr:
		walkExpr(v, n.X)
	case *VariantLit:
		walkExpr(v, n.Enum)
		if n.Variant != nil {
			Walk(v, n.Variant)
		}
		walkExprList(v, n.Args)
	case *LabeledArg:
		if n.Label != nil {
			Walk(v, n.Label)
		}
		walkExpr(v, n.Value)
	case *SpreadElement:
		walkExpr(v, n.X)

	// Specs.
	case *ImportSpec:
		if n.Name != nil {
			Walk(v, n.Name)
		}
		if n.Path != nil {
			Walk(v, n.Path)
		}
	case *ValueSpec:
		walkIdentList(v, n.Names)
		walkExpr(v, n.Type)
		walkExprList(v, n.Values)
	case *TypeSpec:
		if n.Name != nil {
			Walk(v, n.Name)
		}
		if n.TypeParams != nil {
			Walk(v, n.TypeParams)
		}
		walkExpr(v, n.Type)

	// Expressions (leaves first).
	case *Ident, *BasicLit:
		// no children

	case *ParenExpr:
		walkExpr(v, n.X)
	case *UnaryExpr:
		walkExpr(v, n.X)
	case *BinaryExpr:
		walkExpr(v, n.X)
		walkExpr(v, n.Y)
	case *SelectorExpr:
		walkExpr(v, n.X)
		if n.Sel != nil {
			Walk(v, n.Sel)
		}
	case *IndexExpr:
		walkExpr(v, n.X)
		walkExpr(v, n.Index)
	case *IndexListExpr:
		walkExpr(v, n.X)
		walkExprList(v, n.Indices)
	case *SliceExpr:
		walkExpr(v, n.X)
		walkExpr(v, n.Low)
		walkExpr(v, n.High)
		walkExpr(v, n.Max)
	case *CallExpr:
		walkExpr(v, n.Fun)
		walkExprList(v, n.Args)
	case *TypeAssertExpr:
		walkExpr(v, n.X)
		walkExpr(v, n.Type)
	case *StarExpr:
		walkExpr(v, n.X)
	case *KeyValueExpr:
		walkExpr(v, n.Key)
		walkExpr(v, n.Value)
	case *CompositeLit:
		walkExpr(v, n.Type)
		walkExprList(v, n.Elts)
	case *FuncLit:
		if n.Type != nil {
			Walk(v, n.Type)
		}
		if n.Body != nil {
			Walk(v, n.Body)
		}

	// Type expressions.
	case *ArrayType:
		walkExpr(v, n.Len)
		walkExpr(v, n.Elt)
	case *MapType:
		walkExpr(v, n.Key)
		walkExpr(v, n.Value)
	case *StructType:
		if n.Implements != nil {
			Walk(v, n.Implements)
		}
		if n.Fields != nil {
			Walk(v, n.Fields)
		}
	case *InterfaceType:
		if n.Methods != nil {
			Walk(v, n.Methods)
		}
	case *FuncType:
		if n.TypeParams != nil {
			Walk(v, n.TypeParams)
		}
		if n.Params != nil {
			Walk(v, n.Params)
		}
		if n.Results != nil {
			Walk(v, n.Results)
		}
	case *ChanType:
		walkExpr(v, n.Value)
	case *Ellipsis:
		walkExpr(v, n.Elt)

	// Statements.
	case *BlockStmt:
		walkStmtList(v, n.List)
	case *ExprStmt:
		walkExpr(v, n.X)
	case *AssignStmt:
		walkExprList(v, n.Lhs)
		walkExprList(v, n.Rhs)
	case *IncDecStmt:
		walkExpr(v, n.X)
	case *ReturnStmt:
		walkExprList(v, n.Results)
	case *IfStmt:
		if n.Init != nil {
			Walk(v, n.Init)
		}
		walkExpr(v, n.Cond)
		if n.Body != nil {
			Walk(v, n.Body)
		}
		if n.Else != nil {
			Walk(v, n.Else)
		}
	case *ForStmt:
		if n.Init != nil {
			Walk(v, n.Init)
		}
		walkExpr(v, n.Cond)
		if n.Post != nil {
			Walk(v, n.Post)
		}
		if n.Body != nil {
			Walk(v, n.Body)
		}
	case *RangeStmt:
		walkExpr(v, n.Key)
		walkExpr(v, n.Value)
		walkExpr(v, n.X)
		if n.Body != nil {
			Walk(v, n.Body)
		}
	case *SwitchStmt:
		if n.Init != nil {
			Walk(v, n.Init)
		}
		walkExpr(v, n.Tag)
		if n.Body != nil {
			Walk(v, n.Body)
		}
	case *CaseClause:
		walkExprList(v, n.List)
		walkStmtList(v, n.Body)
	case *TypeSwitchStmt:
		if n.Init != nil {
			Walk(v, n.Init)
		}
		if n.Assign != nil {
			Walk(v, n.Assign)
		}
		if n.Body != nil {
			Walk(v, n.Body)
		}
	case *SelectStmt:
		if n.Body != nil {
			Walk(v, n.Body)
		}
	case *CommClause:
		if n.Comm != nil {
			Walk(v, n.Comm)
		}
		walkStmtList(v, n.Body)
	case *SendStmt:
		walkExpr(v, n.Chan)
		walkExpr(v, n.Value)
	case *LabeledStmt:
		if n.Label != nil {
			Walk(v, n.Label)
		}
		if n.Stmt != nil {
			Walk(v, n.Stmt)
		}
	case *DeferStmt:
		if n.Call != nil {
			Walk(v, n.Call)
		}
	case *GoStmt:
		if n.Call != nil {
			Walk(v, n.Call)
		}
	case *BranchStmt:
		if n.Label != nil {
			Walk(v, n.Label)
		}
	case *DeclStmt:
		if n.Decl != nil {
			Walk(v, n.Decl)
		}
	case *EmptyStmt:
		// no children
	case *AssertStmt:
		walkExpr(v, n.Cond)
		walkExpr(v, n.Msg)
		walkExprList(v, n.Args)
	}

	v.Visit(nil)
}

// walkExpr walks an optional expression child, skipping a nil interface.
func walkExpr(v Visitor, x Expr) {
	if x != nil {
		Walk(v, x)
	}
}

// walkExprList walks a slice of expressions.
func walkExprList(v Visitor, list []Expr) {
	for _, x := range list {
		if x != nil {
			Walk(v, x)
		}
	}
}

// walkIdentList walks a slice of identifiers.
func walkIdentList(v Visitor, list []*Ident) {
	for _, id := range list {
		if id != nil {
			Walk(v, id)
		}
	}
}

// walkStmtList walks a slice of statements.
func walkStmtList(v Visitor, list []Stmt) {
	for _, s := range list {
		if s != nil {
			Walk(v, s)
		}
	}
}
