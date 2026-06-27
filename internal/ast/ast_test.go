package ast

import (
	"fmt"
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

// TestWalkGoalDeclChildren asserts Walk descends into the children of each goal
// declaration node added in US-015 (EnumDecl/Variant/PayloadField,
// SealedInterfaceDecl, ImplementsClause) and that a from/derive FuncDecl carries
// its modifier and reports the modifier keyword as its start position.
func TestWalkGoalDeclChildren(t *testing.T) {
	// collect walks root and returns how many times each non-nil node was visited.
	collect := func(root Node) map[Node]int {
		c := &collector{visits: make(map[Node]int)}
		Walk(c, root)
		return c.visits
	}
	// assertChildren asserts parent and each listed child were each visited once.
	assertChildren := func(t *testing.T, visits map[Node]int, parent Node, children ...Node) {
		t.Helper()
		if got := visits[parent]; got != 1 {
			t.Errorf("parent %T visited %d times, want 1", parent, got)
		}
		for _, ch := range children {
			if got := visits[ch]; got != 1 {
				t.Errorf("Walk did not descend from %T into child %T (visited %d times, want 1)", parent, ch, got)
			}
		}
	}

	// enum Decision { Admit; Reject { reason: Rejection } }
	admit := &Variant{Name: &Ident{Name: "Admit"}}
	reasonType := &Ident{Name: "Rejection"}
	reasonField := &PayloadField{Name: &Ident{Name: "reason"}, Type: reasonType}
	reject := &Variant{
		Name:    &Ident{Name: "Reject"},
		Lbrace:  token.Pos{Offset: 1, Line: 1, Col: 2},
		Payload: []*PayloadField{reasonField},
		Rbrace:  token.Pos{Offset: 2, Line: 1, Col: 3},
	}
	enumName := &Ident{Name: "Decision"}
	enum := &EnumDecl{Name: enumName, Variants: []*Variant{admit, reject}}
	{
		visits := collect(enum)
		assertChildren(t, visits, enum, enumName, admit, reject)
		assertChildren(t, visits, admit, admit.Name)
		assertChildren(t, visits, reject, reject.Name, reasonField)
		assertChildren(t, visits, reasonField, reasonField.Name, reasonType)
	}

	// sealed interface Shape { Area() float64 }
	areaField := &Field{Names: []*Ident{{Name: "Area"}}, Type: &FuncType{}}
	methods := &FieldList{List: []*Field{areaField}}
	shapeName := &Ident{Name: "Shape"}
	sealed := &SealedInterfaceDecl{Name: shapeName, Methods: methods}
	{
		visits := collect(sealed)
		assertChildren(t, visits, sealed, shapeName, methods, areaField)
	}

	// type Circle struct implements geom.Shape { Radius float64 }
	ifaceType := &SelectorExpr{X: &Ident{Name: "geom"}, Sel: &Ident{Name: "Shape"}}
	impl := &ImplementsClause{Type: ifaceType}
	radius := &Field{Names: []*Ident{{Name: "Radius"}}, Type: &Ident{Name: "float64"}}
	structType := &StructType{Implements: impl, Fields: &FieldList{List: []*Field{radius}}}
	{
		visits := collect(structType)
		assertChildren(t, visits, structType, impl, structType.Fields)
		assertChildren(t, visits, impl, ifaceType)
	}

	// from func uuidToString(u UUID) string { ... }
	fromName := &Ident{Name: "uuidToString"}
	modPos := token.Pos{Offset: 0, Line: 1, Col: 1}
	fromFunc := &FuncDecl{Mod: FuncFrom, ModPos: modPos, Name: fromName, Type: &FuncType{}}
	{
		visits := collect(fromFunc)
		assertChildren(t, visits, fromFunc, fromName)
		if fromFunc.Mod != FuncFrom {
			t.Errorf("from-func Mod = %v, want FuncFrom", fromFunc.Mod)
		}
		if fromFunc.Pos() != modPos {
			t.Errorf("from-func Pos() = %v, want ModPos %v", fromFunc.Pos(), modPos)
		}
	}

	// derive func fromStorage(s StoredEvent) (EventExecution, error)  — bodyless
	deriveName := &Ident{Name: "fromStorage"}
	deriveFunc := &FuncDecl{Mod: FuncDerive, ModPos: modPos, Name: deriveName, Type: &FuncType{}}
	{
		visits := collect(deriveFunc)
		assertChildren(t, visits, deriveFunc, deriveName)
		if deriveFunc.Mod != FuncDerive {
			t.Errorf("derive-func Mod = %v, want FuncDerive", deriveFunc.Mod)
		}
		if deriveFunc.Pos() != modPos {
			t.Errorf("derive-func Pos() = %v, want ModPos %v", deriveFunc.Pos(), modPos)
		}
	}

	// A plain FuncDecl must NOT report ModPos as its start position.
	plainName := &Ident{Name: "plain"}
	plainFunc := &FuncDecl{Name: plainName, Type: &FuncType{Func: token.Pos{Offset: 5, Line: 1, Col: 6}}}
	if plainFunc.Mod != FuncPlain {
		t.Errorf("plain-func Mod = %v, want FuncPlain", plainFunc.Mod)
	}
	if plainFunc.Pos() != plainFunc.Type.Pos() {
		t.Errorf("plain-func Pos() = %v, want Type.Pos() %v", plainFunc.Pos(), plainFunc.Type.Pos())
	}
}

// TestWalkGoalExprChildren asserts Walk descends into the children of each goal
// expression/pattern node added in US-016 (MatchExpr/MatchArm,
// VariantPattern/RestPattern, UnwrapExpr, VariantLit/LabeledArg, SpreadElement)
// and that a construction VariantLit and a destructuring VariantPattern of the
// same surface shape are distinct node types — the structural fix for the
// Match-before-Enums ordering hack.
func TestWalkGoalExprChildren(t *testing.T) {
	// collect walks root and returns how many times each non-nil node was visited.
	collect := func(root Node) map[Node]int {
		c := &collector{visits: make(map[Node]int)}
		Walk(c, root)
		return c.visits
	}
	// assertChildren asserts parent and each listed child were each visited once.
	assertChildren := func(t *testing.T, visits map[Node]int, parent Node, children ...Node) {
		t.Helper()
		if got := visits[parent]; got != 1 {
			t.Errorf("parent %T visited %d times, want 1", parent, got)
		}
		for _, ch := range children {
			if got := visits[ch]; got != 1 {
				t.Errorf("Walk did not descend from %T into child %T (visited %d times, want 1)", parent, ch, got)
			}
		}
	}

	// Construction: Status.Active(since: now())
	litEnum := &Ident{Name: "Status"}
	litVariant := &Ident{Name: "Active"}
	nowCall := &CallExpr{Fun: &Ident{Name: "now"}}
	sinceArg := &LabeledArg{Label: &Ident{Name: "since"}, Value: nowCall}
	variantLit := &VariantLit{
		Enum:    litEnum,
		Variant: litVariant,
		Lparen:  token.Pos{Offset: 13, Line: 1, Col: 14},
		Args:    []Expr{sinceArg},
		Rparen:  token.Pos{Offset: 25, Line: 1, Col: 26},
	}
	{
		visits := collect(variantLit)
		assertChildren(t, visits, variantLit, litEnum, litVariant, sinceArg)
		assertChildren(t, visits, sinceArg, sinceArg.Label, nowCall)
	}

	// Destructuring pattern: Status.Active(a)
	patEnum := &Ident{Name: "Status"}
	patVariant := &Ident{Name: "Active"}
	binding := &Ident{Name: "a"}
	variantPattern := &VariantPattern{
		Enum:    patEnum,
		Variant: patVariant,
		Lparen:  token.Pos{Offset: 13, Line: 1, Col: 14},
		Binding: binding,
		Rparen:  token.Pos{Offset: 15, Line: 1, Col: 16},
	}
	{
		visits := collect(variantPattern)
		assertChildren(t, visits, variantPattern, patEnum, patVariant, binding)
	}

	// A construction VariantLit and a destructuring VariantPattern of the same
	// surface shape MUST be distinct node types.
	litType := fmt.Sprintf("%T", variantLit)
	patType := fmt.Sprintf("%T", variantPattern)
	if litType == patType {
		t.Fatalf("VariantLit and VariantPattern share a node type (%s); they must be distinct", litType)
	}
	if litType != "*ast.VariantLit" {
		t.Errorf("construction node type = %s, want *ast.VariantLit", litType)
	}
	if patType != "*ast.VariantPattern" {
		t.Errorf("destructuring node type = %s, want *ast.VariantPattern", patType)
	}

	// match s { Status.Active(a) => render(a); _ => fallback() }
	renderCall := &CallExpr{Fun: &Ident{Name: "render"}, Args: []Expr{&Ident{Name: "a"}}}
	arm1 := &MatchArm{Pattern: variantPattern, Arrow: token.Pos{Offset: 16, Line: 1, Col: 17}, Body: renderCall}
	rest := &RestPattern{Underscore: token.Pos{Offset: 30, Line: 2, Col: 1}}
	fallbackCall := &CallExpr{Fun: &Ident{Name: "fallback"}}
	arm2 := &MatchArm{Pattern: rest, Arrow: token.Pos{Offset: 32, Line: 2, Col: 3}, Body: fallbackCall}
	subject := &Ident{Name: "s"}
	matchExpr := &MatchExpr{
		Match:   token.Pos{Offset: 0, Line: 1, Col: 1},
		Subject: subject,
		Lbrace:  token.Pos{Offset: 8, Line: 1, Col: 9},
		Arms:    []*MatchArm{arm1, arm2},
		Rbrace:  token.Pos{Offset: 45, Line: 3, Col: 1},
	}
	{
		visits := collect(matchExpr)
		assertChildren(t, visits, matchExpr, subject, arm1, arm2)
		assertChildren(t, visits, arm1, variantPattern, renderCall)
		assertChildren(t, visits, arm2, rest, fallbackCall)
		// The VariantPattern's own children still walk from inside the arm.
		assertChildren(t, visits, variantPattern, patEnum, patVariant, binding)
		// RestPattern is a leaf — visited, but has no children to descend into.
		if got := visits[rest]; got != 1 {
			t.Errorf("RestPattern visited %d times, want 1", got)
		}
	}

	// Postfix unwrap: g()?
	gCall := &CallExpr{Fun: &Ident{Name: "g"}}
	unwrap := &UnwrapExpr{X: gCall, Question: token.Pos{Offset: 3, Line: 1, Col: 4}}
	{
		visits := collect(unwrap)
		assertChildren(t, visits, unwrap, gCall)
	}

	// Spread element: ...defaults
	defaultsIdent := &Ident{Name: "defaults"}
	spread := &SpreadElement{Ellipsis: token.Pos{Offset: 0, Line: 1, Col: 1}, X: defaultsIdent}
	{
		visits := collect(spread)
		assertChildren(t, visits, spread, defaultsIdent)
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
