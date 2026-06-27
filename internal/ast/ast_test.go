package ast

import (
	"testing"

	"goal/internal/token"
)

// collector records every non-nil node Walk passes to Visit. Returning itself
// keeps the traversal descending; nil end-markers (Visit(nil)) are ignored.
type collector struct {
	visits map[Node]int
	order  []Node
}

func (c *collector) Visit(node Node) Visitor {
	if node == nil {
		return c
	}
	c.visits[node]++
	c.order = append(c.order, node)
	return c
}

// TestWalkVisitsEveryNodeExactlyOnce builds a representative AST tree by hand,
// collecting every constructed node, then asserts Walk visits each of those
// nodes exactly once and visits nothing else.
func TestWalkVisitsEveryNodeExactlyOnce(t *testing.T) {
	var want []Node // every node constructed, in construction order
	// add registers a node as expected and returns it unchanged.
	add := func(n Node) Node {
		want = append(want, n)
		return n
	}

	p := func(off int) token.Pos { return token.Pos{Offset: off, Line: 1, Col: off + 1} }

	// Helpers that both construct and register each node.
	ident := func(name string) *Ident { return add(&Ident{Name: name}).(*Ident) }
	lit := func(kind token.Kind, val string) *BasicLit {
		return add(&BasicLit{Kind: kind, Value: val}).(*BasicLit)
	}

	// package p
	pkgName := ident("p")

	// import f "fmt"
	importSpec := add(&ImportSpec{
		Name: ident("f"),
		Path: lit(token.STRING, `"fmt"`),
	}).(*ImportSpec)
	importDecl := add(&GenDecl{Tok: token.IMPORT, Specs: []Spec{importSpec}}).(*GenDecl)

	// type Point struct { X, Y int }
	structFields := add(&FieldList{List: []*Field{
		add(&Field{
			Names: []*Ident{ident("X"), ident("Y")},
			Type:  ident("int"),
		}).(*Field),
	}}).(*FieldList)
	structType := add(&StructType{Fields: structFields}).(*StructType)
	typeSpec := add(&TypeSpec{Name: ident("Point"), Type: structType}).(*TypeSpec)
	typeDecl := add(&GenDecl{Tok: token.TYPE, Specs: []Spec{typeSpec}}).(*GenDecl)

	// var data []map[string]*Point
	mapType := add(&MapType{Key: ident("string"), Value: add(&StarExpr{X: ident("Point")}).(*StarExpr)}).(*MapType)
	sliceType := add(&ArrayType{Elt: mapType}).(*ArrayType)
	valueSpec := add(&ValueSpec{Names: []*Ident{ident("data")}, Type: sliceType}).(*ValueSpec)
	varDecl := add(&GenDecl{Tok: token.VAR, Specs: []Spec{valueSpec}}).(*GenDecl)

	// func (r *Point) Add(a int, rest ...string) (int, error) { ... }
	recv := add(&FieldList{List: []*Field{
		add(&Field{Names: []*Ident{ident("r")}, Type: add(&StarExpr{X: ident("Point")}).(*StarExpr)}).(*Field),
	}}).(*FieldList)
	params := add(&FieldList{List: []*Field{
		add(&Field{Names: []*Ident{ident("a")}, Type: ident("int")}).(*Field),
		add(&Field{Names: []*Ident{ident("rest")}, Type: add(&Ellipsis{Elt: ident("string")}).(*Ellipsis)}).(*Field),
	}}).(*FieldList)
	results := add(&FieldList{List: []*Field{
		add(&Field{Type: ident("int")}).(*Field),
		add(&Field{Type: ident("error")}).(*Field),
	}}).(*FieldList)
	funcType := add(&FuncType{Params: params, Results: results}).(*FuncType)

	// Body statements:
	//   sum := r.X + a          (AssignStmt: SelectorExpr, Ident; BinaryExpr)
	assign := add(&AssignStmt{
		Lhs: []Expr{ident("sum")},
		Tok: token.DEFINE,
		Rhs: []Expr{
			add(&BinaryExpr{
				X:  add(&SelectorExpr{X: ident("r"), Sel: ident("X")}).(*SelectorExpr),
				Op: token.ADD,
				Y:  ident("a"),
			}).(*BinaryExpr),
		},
	}).(*AssignStmt)

	//   sum++                   (IncDecStmt)
	incdec := add(&IncDecStmt{X: ident("sum"), Tok: token.INC}).(*IncDecStmt)

	//   if sum > 0 { defer f.Println(sum) } else { sum = -sum }
	deferCall := add(&CallExpr{
		Fun:  add(&SelectorExpr{X: ident("f"), Sel: ident("Println")}).(*SelectorExpr),
		Args: []Expr{ident("sum")},
	}).(*CallExpr)
	deferStmt := add(&DeferStmt{Call: deferCall}).(*DeferStmt)
	ifBody := add(&BlockStmt{List: []Stmt{deferStmt}}).(*BlockStmt)
	elseAssign := add(&AssignStmt{
		Lhs: []Expr{ident("sum")},
		Tok: token.ASSIGN,
		Rhs: []Expr{add(&UnaryExpr{Op: token.SUB, X: ident("sum")}).(*UnaryExpr)},
	}).(*AssignStmt)
	elseBlock := add(&BlockStmt{List: []Stmt{elseAssign}}).(*BlockStmt)
	ifStmt := add(&IfStmt{
		Cond: add(&BinaryExpr{X: ident("sum"), Op: token.GTR, Y: lit(token.INT, "0")}).(*BinaryExpr),
		Body: ifBody,
		Else: elseBlock,
	}).(*IfStmt)

	//   for i := 0; i < 3; i++ { _ = data[i] }
	forInit := add(&AssignStmt{Lhs: []Expr{ident("i")}, Tok: token.DEFINE, Rhs: []Expr{lit(token.INT, "0")}}).(*AssignStmt)
	forCond := add(&BinaryExpr{X: ident("i"), Op: token.LSS, Y: lit(token.INT, "3")}).(*BinaryExpr)
	forPost := add(&IncDecStmt{X: ident("i"), Tok: token.INC}).(*IncDecStmt)
	forBodyStmt := add(&AssignStmt{
		Lhs: []Expr{ident("_")},
		Tok: token.ASSIGN,
		Rhs: []Expr{add(&IndexExpr{X: ident("data"), Index: ident("i")}).(*IndexExpr)},
	}).(*AssignStmt)
	forBody := add(&BlockStmt{List: []Stmt{forBodyStmt}}).(*BlockStmt)
	forStmt := add(&ForStmt{Init: forInit, Cond: forCond, Post: forPost, Body: forBody}).(*ForStmt)

	//   return sum, nil
	returnStmt := add(&ReturnStmt{Results: []Expr{ident("sum"), ident("nil")}}).(*ReturnStmt)

	funcBody := add(&BlockStmt{List: []Stmt{assign, incdec, ifStmt, forStmt, returnStmt}}).(*BlockStmt)
	funcDecl := add(&FuncDecl{Recv: recv, Name: ident("Add"), Type: funcType, Body: funcBody}).(*FuncDecl)

	file := add(&File{
		Package: p(0),
		Name:    pkgName,
		Imports: []*ImportSpec{importSpec},
		Decls:   []Decl{importDecl, typeDecl, varDecl, funcDecl},
	}).(*File)

	// Walk and collect.
	c := &collector{visits: make(map[Node]int)}
	Walk(c, file)

	// Every constructed node must be visited exactly once.
	if len(c.order) != len(want) {
		t.Fatalf("Walk visited %d nodes, want %d constructed nodes", len(c.order), len(want))
	}
	for _, n := range want {
		switch got := c.visits[n]; got {
		case 1:
			// good
		case 0:
			t.Errorf("node %T was never visited", n)
		default:
			t.Errorf("node %T was visited %d times, want exactly 1", n, got)
		}
	}
	// And Walk must not visit anything that was not constructed.
	for n := range c.visits {
		if !contains(want, n) {
			t.Errorf("Walk visited an unexpected node %T", n)
		}
	}
}

// TestWalkNilNodeIsNoop asserts Walk on a nil node neither panics nor visits.
func TestWalkNilNodeIsNoop(t *testing.T) {
	c := &collector{visits: make(map[Node]int)}
	Walk(c, nil)
	if len(c.order) != 0 {
		t.Fatalf("Walk(v, nil) visited %d nodes, want 0", len(c.order))
	}
}

func contains(list []Node, target Node) bool {
	for _, n := range list {
		if n == target {
			return true
		}
	}
	return false
}
