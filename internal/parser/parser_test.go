package parser

import (
	"fmt"
	"testing"

	"goal/internal/ast"
	"goal/internal/token"
)

// sampleSrc exercises every declaration form in scope for US-017: the package
// clause; single and grouped imports (incl. named, blank, and dot); single and
// grouped const/var/type (incl. a struct, an interface, a type alias, and a
// composite-literal initializer); and a plain function plus a method.
const sampleSrc = `package sample

import "fmt"

import (
	"strings"
	m "math"
	_ "embed"
	. "errors"
)

const Pi = 3

const (
	A = 1
	B = 2
)

var greeting = "hi"

var (
	count int
	name  string = "x"
)

var origin = Point{X: 1, Y: 2}

type ID struct {
	v string
}

type (
	Celsius = float64
	Point   struct {
		X int
		Y int
	}
)

type Stringer interface {
	String() string
}

func plain(x int, y string) int { return x }

func (id ID) Value() string { return id.v }
`

func TestParseFileDeclarationShape(t *testing.T) {
	file, err := ParseFile(sampleSrc)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}

	// Package clause.
	if file.Name == nil || file.Name.Name != "sample" {
		t.Fatalf("package name = %v, want sample", file.Name)
	}
	if file.Package.Line != 1 {
		t.Errorf("package keyword line = %d, want 1", file.Package.Line)
	}

	// Imports are reachable from the file's import list (across both import decls).
	type imp struct{ name, path string }
	wantImports := []imp{
		{"", `"fmt"`},
		{"", `"strings"`},
		{"m", `"math"`},
		{"_", `"embed"`},
		{".", `"errors"`},
	}
	if len(file.Imports) != len(wantImports) {
		t.Fatalf("len(Imports) = %d, want %d", len(file.Imports), len(wantImports))
	}
	for i, w := range wantImports {
		got := file.Imports[i]
		gotName := ""
		if got.Name != nil {
			gotName = got.Name.Name
		}
		if gotName != w.name {
			t.Errorf("import[%d] name = %q, want %q", i, gotName, w.name)
		}
		if got.Path == nil || got.Path.Value != w.path {
			t.Errorf("import[%d] path = %v, want %q", i, got.Path, w.path)
		}
	}

	// Declaration list shape: kinds, GenDecl tokens, and spec names in order.
	if len(file.Decls) != 12 {
		t.Fatalf("len(Decls) = %d, want 12", len(file.Decls))
	}

	wantGen := []struct {
		idx       int
		tok       token.Kind
		specNames [][]string // names per spec (TypeSpec/ValueSpec use one entry per spec)
	}{
		{0, token.IMPORT, nil},
		{1, token.IMPORT, nil},
		{2, token.CONST, [][]string{{"Pi"}}},
		{3, token.CONST, [][]string{{"A"}, {"B"}}},
		{4, token.VAR, [][]string{{"greeting"}}},
		{5, token.VAR, [][]string{{"count"}, {"name"}}},
		{6, token.VAR, [][]string{{"origin"}}},
		{7, token.TYPE, [][]string{{"ID"}}},
		{8, token.TYPE, [][]string{{"Celsius"}, {"Point"}}},
		{9, token.TYPE, [][]string{{"Stringer"}}},
	}
	for _, w := range wantGen {
		gd, ok := file.Decls[w.idx].(*ast.GenDecl)
		if !ok {
			t.Fatalf("Decls[%d] is %T, want *ast.GenDecl", w.idx, file.Decls[w.idx])
		}
		if gd.Tok != w.tok {
			t.Errorf("Decls[%d].Tok = %s, want %s", w.idx, gd.Tok, w.tok)
		}
		if w.specNames == nil {
			continue
		}
		if len(gd.Specs) != len(w.specNames) {
			t.Fatalf("Decls[%d] has %d specs, want %d", w.idx, len(gd.Specs), len(w.specNames))
		}
		for j, names := range w.specNames {
			gotNames := specNames(t, gd.Specs[j])
			if len(gotNames) != len(names) {
				t.Fatalf("Decls[%d].Specs[%d] has %d names, want %d", w.idx, j, len(gotNames), len(names))
			}
			for k, n := range names {
				if gotNames[k] != n {
					t.Errorf("Decls[%d].Specs[%d] name[%d] = %q, want %q", w.idx, j, k, gotNames[k], n)
				}
			}
		}
	}

	// The last two decls are functions.
	plain, ok := file.Decls[10].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("Decls[10] is %T, want *ast.FuncDecl", file.Decls[10])
	}
	if plain.Name.Name != "plain" {
		t.Errorf("func name = %q, want plain", plain.Name.Name)
	}
	if plain.Recv != nil {
		t.Errorf("plain has receiver, want none")
	}
	if got := len(plain.Type.Params.List); got != 2 {
		t.Errorf("plain params = %d, want 2", got)
	}
	if plain.Type.Results == nil || len(plain.Type.Results.List) != 1 {
		t.Errorf("plain results = %v, want 1", plain.Type.Results)
	}
	if plain.Body == nil {
		t.Errorf("plain body is nil, want a (skipped) block")
	}

	method, ok := file.Decls[11].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("Decls[11] is %T, want *ast.FuncDecl", file.Decls[11])
	}
	if method.Name.Name != "Value" {
		t.Errorf("method name = %q, want Value", method.Name.Name)
	}
	if method.Recv == nil || len(method.Recv.List) != 1 {
		t.Fatalf("method receiver = %v, want 1 field", method.Recv)
	}
	if recvNames := method.Recv.List[0].Names; len(recvNames) != 1 || recvNames[0].Name != "id" {
		t.Errorf("method receiver name = %v, want id", recvNames)
	}
}

// specNames extracts the declared names from a value or type spec.
func specNames(t *testing.T, s ast.Spec) []string {
	t.Helper()
	switch sp := s.(type) {
	case *ast.ValueSpec:
		out := make([]string, len(sp.Names))
		for i, n := range sp.Names {
			out[i] = n.Name
		}
		return out
	case *ast.TypeSpec:
		return []string{sp.Name.Name}
	default:
		t.Fatalf("unexpected spec type %T", s)
		return nil
	}
}

func TestParseFileStructAndAlias(t *testing.T) {
	file, err := ParseFile(sampleSrc)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}

	// type ID struct { v string }
	idSpec := file.Decls[7].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
	st, ok := idSpec.Type.(*ast.StructType)
	if !ok {
		t.Fatalf("ID underlying type is %T, want *ast.StructType", idSpec.Type)
	}
	if len(st.Fields.List) != 1 || len(st.Fields.List[0].Names) != 1 || st.Fields.List[0].Names[0].Name != "v" {
		t.Errorf("ID struct fields = %v, want one field 'v'", st.Fields.List)
	}

	// grouped type: Celsius = float64 (alias), Point struct{...}
	grp := file.Decls[8].(*ast.GenDecl)
	celsius := grp.Specs[0].(*ast.TypeSpec)
	if id, ok := celsius.Type.(*ast.Ident); !ok || id.Name != "float64" {
		t.Errorf("Celsius alias type = %v, want float64", celsius.Type)
	}
	point := grp.Specs[1].(*ast.TypeSpec)
	pst, ok := point.Type.(*ast.StructType)
	if !ok {
		t.Fatalf("Point underlying type is %T, want *ast.StructType", point.Type)
	}
	if len(pst.Fields.List) != 2 {
		t.Errorf("Point has %d fields, want 2", len(pst.Fields.List))
	}

	// interface Stringer with one method
	ifaceSpec := file.Decls[9].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
	it, ok := ifaceSpec.Type.(*ast.InterfaceType)
	if !ok {
		t.Fatalf("Stringer underlying type is %T, want *ast.InterfaceType", ifaceSpec.Type)
	}
	if len(it.Methods.List) != 1 || it.Methods.List[0].Names[0].Name != "String" {
		t.Errorf("Stringer methods = %v, want one method 'String'", it.Methods.List)
	}
}

func TestParseFileCompositeInitializer(t *testing.T) {
	file, err := ParseFile(sampleSrc)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}
	// var origin = Point{X: 1, Y: 2}
	originSpec := file.Decls[6].(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
	if len(originSpec.Values) != 1 {
		t.Fatalf("origin has %d values, want 1", len(originSpec.Values))
	}
	cl, ok := originSpec.Values[0].(*ast.CompositeLit)
	if !ok {
		t.Fatalf("origin value is %T, want *ast.CompositeLit", originSpec.Values[0])
	}
	if id, ok := cl.Type.(*ast.Ident); !ok || id.Name != "Point" {
		t.Errorf("composite type = %v, want Point", cl.Type)
	}
	if len(cl.Elts) != 2 {
		t.Fatalf("composite has %d elements, want 2", len(cl.Elts))
	}
	kv, ok := cl.Elts[0].(*ast.KeyValueExpr)
	if !ok {
		t.Fatalf("composite elt[0] is %T, want *ast.KeyValueExpr", cl.Elts[0])
	}
	if key, ok := kv.Key.(*ast.Ident); !ok || key.Name != "X" {
		t.Errorf("composite elt[0] key = %v, want X", kv.Key)
	}
}

// TestParseInterfaceMethodBoundaries pins the result-line boundary that lets a
// void method coexist with later elements. Because the lexer strips newlines, a
// method spec ends only because its result must sit on the same line as the
// closing ')'; otherwise the next element's name is swallowed as a return type.
func TestParseInterfaceMethodBoundaries(t *testing.T) {
	const src = `package p

type R interface {
	Reset()
	Area() float64
	Reader
}
`
	file, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}

	it := file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.InterfaceType)
	if len(it.Methods.List) != 3 {
		t.Fatalf("interface has %d elements, want 3 (Reset, Area, Reader)", len(it.Methods.List))
	}

	// Reset: a void method — named, with an empty result set.
	reset := it.Methods.List[0]
	if len(reset.Names) != 1 || reset.Names[0].Name != "Reset" {
		t.Fatalf("element 0 names = %v, want [Reset]", reset.Names)
	}
	if ft := reset.Type.(*ast.FuncType); ft.Results != nil && len(ft.Results.List) != 0 {
		t.Errorf("Reset has results %v, want none", ft.Results)
	}

	// Area: a returning method — its result type must NOT have absorbed the
	// embedded Reader on the following line.
	area := it.Methods.List[1]
	if len(area.Names) != 1 || area.Names[0].Name != "Area" {
		t.Fatalf("element 1 names = %v, want [Area]", area.Names)
	}
	areaFt := area.Type.(*ast.FuncType)
	if areaFt.Results == nil || len(areaFt.Results.List) != 1 {
		t.Fatalf("Area results = %v, want one result", areaFt.Results)
	}
	if id, ok := areaFt.Results.List[0].Type.(*ast.Ident); !ok || id.Name != "float64" {
		t.Errorf("Area result type = %v, want float64", areaFt.Results.List[0].Type)
	}

	// Reader: an embedded interface — unnamed, the type itself is the name.
	reader := it.Methods.List[2]
	if len(reader.Names) != 0 {
		t.Fatalf("embedded element names = %v, want none", reader.Names)
	}
	if id, ok := reader.Type.(*ast.Ident); !ok || id.Name != "Reader" {
		t.Errorf("embedded element type = %v, want Ident Reader", reader.Type)
	}
}

func TestParseFileErrors(t *testing.T) {
	cases := map[string]string{
		"missing package name": "package",
		"stray top-level token": "package p\n@",
		"bad import path":       "package p\nimport fmt",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseFile(src); err == nil {
				t.Errorf("ParseFile(%q) returned nil error, want non-nil", src)
			}
		})
	}
}

func TestParseFileWellFormedNoError(t *testing.T) {
	src := "package p\nfunc f() {}\n"
	file, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}
	if len(file.Decls) != 1 {
		t.Fatalf("len(Decls) = %d, want 1", len(file.Decls))
	}
	if _, ok := file.Decls[0].(*ast.FuncDecl); !ok {
		t.Errorf("Decls[0] is %T, want *ast.FuncDecl", file.Decls[0])
	}
}

// stmtSrc exercises every statement form in scope for US-018 inside one function
// body: short-var and ordinary assignment, a const/var declaration, a nested
// block, an if/else-if/else chain, all three for shapes (three-clause,
// condition-only, and range), an expression switch, defer, go, increment, and
// return. The standalone block is placed after a literal-terminated statement so
// the body "{" is not misread as a composite literal.
const stmtSrc = `package sample

func body(n int) int {
	x := 0
	{
		z := 1
		x = z
	}
	var y int
	y = n
	if n {
		x = 1
	} else if y {
		x = 2
	} else {
		x = 3
	}
	for i := 0; i; i++ {
		x = i
	}
	for n {
		break
	}
	for k, v := range items {
		x = v
		continue
	}
	switch x {
	case 1:
		x = 10
	default:
		x = 0
	}
	defer cleanup()
	go run()
	x++
	return x
}
`

func TestParseFunctionBodyStatements(t *testing.T) {
	file, err := ParseFile(stmtSrc)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}
	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("Decls[0] is %T, want *ast.FuncDecl", file.Decls[0])
	}
	if fn.Body == nil {
		t.Fatal("function body is nil")
	}
	list := fn.Body.List

	// The statement list shape, in order.
	wantKinds := []string{
		"*ast.AssignStmt", // x := 0
		"*ast.BlockStmt",  // { ... }
		"*ast.DeclStmt",   // var y int
		"*ast.AssignStmt", // y = n
		"*ast.IfStmt",     // if/else-if/else
		"*ast.ForStmt",    // for i := 0; i; i++
		"*ast.ForStmt",    // for n
		"*ast.RangeStmt",  // for k, v := range items
		"*ast.SwitchStmt", // switch x
		"*ast.DeferStmt",  // defer cleanup()
		"*ast.GoStmt",     // go run()
		"*ast.IncDecStmt", // x++
		"*ast.ReturnStmt", // return x
	}
	if len(list) != len(wantKinds) {
		t.Fatalf("body has %d statements, want %d", len(list), len(wantKinds))
	}
	for i, want := range wantKinds {
		if got := fmt.Sprintf("%T", list[i]); got != want {
			t.Errorf("stmt[%d] = %s, want %s", i, got, want)
		}
	}

	// 1: short-var declaration uses DEFINE.
	if a := list[0].(*ast.AssignStmt); a.Tok != token.DEFINE {
		t.Errorf("stmt[0] tok = %s, want :=", a.Tok)
	}
	// 2: nested block has its own two statements.
	if b := list[1].(*ast.BlockStmt); len(b.List) != 2 {
		t.Errorf("nested block has %d statements, want 2", len(b.List))
	}
	// 3: declaration statement wraps a var GenDecl.
	if d := list[2].(*ast.DeclStmt); d.Decl.(*ast.GenDecl).Tok != token.VAR {
		t.Errorf("stmt[2] decl tok = %s, want var", d.Decl.(*ast.GenDecl).Tok)
	}
	// 4: ordinary assignment uses ASSIGN.
	if a := list[3].(*ast.AssignStmt); a.Tok != token.ASSIGN {
		t.Errorf("stmt[3] tok = %s, want =", a.Tok)
	}

	// 5: if with a condition and an else-if chain ending in an else block.
	ifs := list[4].(*ast.IfStmt)
	if ifs.Cond == nil {
		t.Error("if has nil condition")
	}
	elseIf, ok := ifs.Else.(*ast.IfStmt)
	if !ok {
		t.Fatalf("if.Else is %T, want *ast.IfStmt", ifs.Else)
	}
	if _, ok := elseIf.Else.(*ast.BlockStmt); !ok {
		t.Errorf("else-if.Else is %T, want *ast.BlockStmt", elseIf.Else)
	}

	// 6: three-clause for has Init, Cond, and Post.
	f3 := list[5].(*ast.ForStmt)
	if f3.Init == nil || f3.Cond == nil || f3.Post == nil {
		t.Errorf("three-clause for: Init=%v Cond=%v Post=%v, want all set", f3.Init, f3.Cond, f3.Post)
	}
	if _, ok := f3.Post.(*ast.IncDecStmt); !ok {
		t.Errorf("three-clause for Post is %T, want *ast.IncDecStmt", f3.Post)
	}

	// 7: condition-only for has a Cond but no Init/Post.
	f1 := list[6].(*ast.ForStmt)
	if f1.Init != nil || f1.Post != nil || f1.Cond == nil {
		t.Errorf("condition-only for: Init=%v Cond=%v Post=%v", f1.Init, f1.Cond, f1.Post)
	}

	// 8: range for binds key and value.
	rng := list[7].(*ast.RangeStmt)
	if rng.Tok != token.DEFINE {
		t.Errorf("range tok = %s, want :=", rng.Tok)
	}
	if k, ok := rng.Key.(*ast.Ident); !ok || k.Name != "k" {
		t.Errorf("range key = %v, want k", rng.Key)
	}
	if v, ok := rng.Value.(*ast.Ident); !ok || v.Name != "v" {
		t.Errorf("range value = %v, want v", rng.Value)
	}
	if x, ok := rng.X.(*ast.Ident); !ok || x.Name != "items" {
		t.Errorf("range X = %v, want items", rng.X)
	}

	// 9: switch with two clauses (a case and a default).
	sw := list[8].(*ast.SwitchStmt)
	if sw.Tag == nil {
		t.Error("switch tag is nil")
	}
	if len(sw.Body.List) != 2 {
		t.Fatalf("switch has %d clauses, want 2", len(sw.Body.List))
	}
	c0 := sw.Body.List[0].(*ast.CaseClause)
	if len(c0.List) != 1 {
		t.Errorf("case[0] has %d expressions, want 1", len(c0.List))
	}
	if def := sw.Body.List[1].(*ast.CaseClause); def.List != nil {
		t.Errorf("case[1] (default) List = %v, want nil", def.List)
	}

	// 10/11: defer and go carry a call expression.
	if d := list[9].(*ast.DeferStmt); d.Call == nil {
		t.Error("defer has nil call")
	}
	if g := list[10].(*ast.GoStmt); g.Call == nil {
		t.Error("go has nil call")
	}
	// 12: increment.
	if inc := list[11].(*ast.IncDecStmt); inc.Tok != token.INC {
		t.Errorf("stmt[11] tok = %s, want ++", inc.Tok)
	}
	// 13: return with one result.
	if r := list[12].(*ast.ReturnStmt); len(r.Results) != 1 {
		t.Errorf("return has %d results, want 1", len(r.Results))
	}
}

// parseExprInBody parses a single expression by wrapping it in a function body
// assignment (`func f() { _ = <expr> }`) and returns the assigned value, so the
// expression flows through the real parseExpr path used everywhere.
func parseExprInBody(t *testing.T, expr string) ast.Expr {
	t.Helper()
	src := "package p\nfunc f() { _ = " + expr + " }"
	file, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile(%q) returned error: %v", expr, err)
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	assign, ok := fn.Body.List[0].(*ast.AssignStmt)
	if !ok {
		t.Fatalf("body[0] is %T, want *ast.AssignStmt", fn.Body.List[0])
	}
	if len(assign.Rhs) != 1 {
		t.Fatalf("assignment has %d rhs values, want 1", len(assign.Rhs))
	}
	return assign.Rhs[0]
}

// TestParseExpressionPrecedence pins the US-019 acceptance criteria: postfix `?`
// parses to UnwrapExpr over its operand, and binary operators nest by Go
// precedence with left associativity, with unary binding tighter than binary.
func TestParseExpressionPrecedence(t *testing.T) {
	// f(x)? -> UnwrapExpr wrapping a CallExpr.
	t.Run("unwrap call", func(t *testing.T) {
		x := parseExprInBody(t, "f(x)?")
		u, ok := x.(*ast.UnwrapExpr)
		if !ok {
			t.Fatalf("got %T, want *ast.UnwrapExpr", x)
		}
		if _, ok := u.X.(*ast.CallExpr); !ok {
			t.Fatalf("unwrap operand is %T, want *ast.CallExpr", u.X)
		}
	})

	// a.b? -> UnwrapExpr wrapping a SelectorExpr.
	t.Run("unwrap selector", func(t *testing.T) {
		x := parseExprInBody(t, "a.b?")
		u, ok := x.(*ast.UnwrapExpr)
		if !ok {
			t.Fatalf("got %T, want *ast.UnwrapExpr", x)
		}
		sel, ok := u.X.(*ast.SelectorExpr)
		if !ok {
			t.Fatalf("unwrap operand is %T, want *ast.SelectorExpr", u.X)
		}
		if sel.Sel.Name != "b" {
			t.Errorf("selector field = %q, want b", sel.Sel.Name)
		}
	})

	// a + b * c == d -> (a + (b * c)) == d.
	t.Run("mixed precedence", func(t *testing.T) {
		x := parseExprInBody(t, "a + b * c == d")
		eq, ok := x.(*ast.BinaryExpr)
		if !ok || eq.Op != token.EQL {
			t.Fatalf("top = %T op %v, want *ast.BinaryExpr op ==", x, opOf(x))
		}
		if name(eq.Y) != "d" {
			t.Errorf("rhs of == = %q, want d", name(eq.Y))
		}
		add, ok := eq.X.(*ast.BinaryExpr)
		if !ok || add.Op != token.ADD {
			t.Fatalf("lhs of == = %T op %v, want *ast.BinaryExpr op +", eq.X, opOf(eq.X))
		}
		if name(add.X) != "a" {
			t.Errorf("lhs of + = %q, want a", name(add.X))
		}
		mul, ok := add.Y.(*ast.BinaryExpr)
		if !ok || mul.Op != token.MUL {
			t.Fatalf("rhs of + = %T op %v, want *ast.BinaryExpr op *", add.Y, opOf(add.Y))
		}
		if name(mul.X) != "b" || name(mul.Y) != "c" {
			t.Errorf("* operands = %q, %q, want b, c", name(mul.X), name(mul.Y))
		}
	})

	// a - b - c -> ((a - b) - c): left associative.
	t.Run("left associative", func(t *testing.T) {
		x := parseExprInBody(t, "a - b - c")
		outer, ok := x.(*ast.BinaryExpr)
		if !ok || outer.Op != token.SUB {
			t.Fatalf("top = %T op %v, want *ast.BinaryExpr op -", x, opOf(x))
		}
		if name(outer.Y) != "c" {
			t.Errorf("rhs of outer - = %q, want c", name(outer.Y))
		}
		inner, ok := outer.X.(*ast.BinaryExpr)
		if !ok || inner.Op != token.SUB {
			t.Fatalf("lhs of outer - = %T, want a nested - expression", outer.X)
		}
		if name(inner.X) != "a" || name(inner.Y) != "b" {
			t.Errorf("inner - operands = %q, %q, want a, b", name(inner.X), name(inner.Y))
		}
	})

	// -a * b -> (-a) * b: unary binds tighter than binary.
	t.Run("unary tighter than binary", func(t *testing.T) {
		x := parseExprInBody(t, "-a * b")
		mul, ok := x.(*ast.BinaryExpr)
		if !ok || mul.Op != token.MUL {
			t.Fatalf("top = %T op %v, want *ast.BinaryExpr op *", x, opOf(x))
		}
		un, ok := mul.X.(*ast.UnaryExpr)
		if !ok || un.Op != token.SUB {
			t.Fatalf("lhs of * = %T, want *ast.UnaryExpr op -", mul.X)
		}
		if name(un.X) != "a" {
			t.Errorf("unary operand = %q, want a", name(un.X))
		}
	})
}

// name returns the identifier name of an *ast.Ident expression, or "" otherwise.
func name(e ast.Expr) string {
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// opOf renders a binary expression's operator for diagnostics, or "?" otherwise.
func opOf(e ast.Expr) string {
	if b, ok := e.(*ast.BinaryExpr); ok {
		return b.Op.String()
	}
	return "?"
}

// TestParseTypeArgList pins the multi-element generic type-argument list grammar
// added for the parse-100%-of-corpus gate: a result type Result[int, error]
// parses with zero errors into an IndexListExpr with two indices, and a nested
// slice type argument Result[[]byte, error] also parses.
func TestParseTypeArgList(t *testing.T) {
	t.Run("two type args", func(t *testing.T) {
		file, err := ParseFile("package p\nfunc f() Result[int, error] { return nil }")
		if err != nil {
			t.Fatalf("ParseFile returned error: %v", err)
		}
		res := file.Decls[0].(*ast.FuncDecl).Type.Results.List[0].Type
		il, ok := res.(*ast.IndexListExpr)
		if !ok {
			t.Fatalf("result type is %T, want *ast.IndexListExpr", res)
		}
		if len(il.Indices) != 2 {
			t.Fatalf("IndexListExpr has %d indices, want 2", len(il.Indices))
		}
	})

	t.Run("slice type arg", func(t *testing.T) {
		if _, err := ParseFile("package p\nfunc f() Result[[]byte, error] { return nil }"); err != nil {
			t.Fatalf("ParseFile returned error: %v", err)
		}
	})
}

// TestParseTypeLiteralOperand pins the type-literal-in-operand grammar: a slice
// conversion []byte(p) parses as a call over an ArrayType, and a map composite
// literal map[string]string{} parses with zero errors.
func TestParseTypeLiteralOperand(t *testing.T) {
	t.Run("slice conversion", func(t *testing.T) {
		x := parseExprInBody(t, "[]byte(p)")
		call, ok := x.(*ast.CallExpr)
		if !ok {
			t.Fatalf("got %T, want *ast.CallExpr", x)
		}
		if _, ok := call.Fun.(*ast.ArrayType); !ok {
			t.Fatalf("call target is %T, want *ast.ArrayType", call.Fun)
		}
	})

	t.Run("map composite literal", func(t *testing.T) {
		x := parseExprInBody(t, "map[string]string{}")
		cl, ok := x.(*ast.CompositeLit)
		if !ok {
			t.Fatalf("got %T, want *ast.CompositeLit", x)
		}
		if _, ok := cl.Type.(*ast.MapType); !ok {
			t.Fatalf("composite type is %T, want *ast.MapType", cl.Type)
		}
	})
}

// TestParseOptionalColonPayloadField pins that an enum variant payload field
// parses both with a colon (`name: Type`) and without (`name Type`).
func TestParseOptionalColonPayloadField(t *testing.T) {
	for _, src := range []string{
		"package p\nenum E { Active { since int } }",
		"package p\nenum E { Active { since: int } }",
	} {
		file, err := ParseFile(src)
		if err != nil {
			t.Fatalf("ParseFile(%q) returned error: %v", src, err)
		}
		enum := file.Decls[0].(*ast.EnumDecl)
		field := enum.Variants[0].Payload[0]
		if field.Name.Name != "since" {
			t.Errorf("payload field name = %q, want since", field.Name.Name)
		}
		if id, ok := field.Type.(*ast.Ident); !ok || id.Name != "int" {
			t.Errorf("payload field type = %T %v, want *ast.Ident int", field.Type, field.Type)
		}
	}
}
