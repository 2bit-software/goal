package parser

import (
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
