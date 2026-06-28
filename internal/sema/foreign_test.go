package sema

import (
	"path/filepath"
	"reflect"
	"testing"

	"goal/internal/parser"
)

// TestEnrichForeignResolvesImportedStructAndMethod is the foreign-enrichment gate: for a
// goal file importing a package with an exported struct AND an exported method,
// sema.EnrichForeign must populate ForeignMethods + Structs from the imported package read
// via the resolver. The fixture (testdata/extpkg) declares an exported struct (Outer/Inner,
// with package-local pointer/slice/map fields) and an exported method (Inner.Close), so the
// assertions exercise both maps. Package-local type references are qualified by the import
// alias (Inner -> ext.Inner).
func TestEnrichForeignResolvesImportedStructAndMethod(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("testdata", "extpkg"))
	if err != nil {
		t.Fatal(err)
	}
	// A `derive func` references ext.Outer, so the AST's import list loads ext; sema then
	// reads the whole foreign package, so the struct AND method sets are both populated.
	src := `package consumer

import ext "example.com/ext"

type Local struct{ id string }

derive func make(o *ext.Outer) Local
`
	resolve := func(importPath, fromDir string) (string, error) {
		if importPath != "example.com/ext" {
			t.Fatalf("unexpected import path %q", importPath)
		}
		return dir, nil
	}

	file, perr := parser.ParseFile(src)
	if perr != nil {
		t.Fatalf("parse goal source: %v", perr)
	}
	info := Resolve(file)
	if errs := EnrichForeign(info, file.Imports, ".", resolve); len(errs) != 0 {
		t.Fatalf("sema.EnrichForeign errors: %v", errs)
	}

	// Structs: the imported Outer is enriched with exactly its exported fields (the
	// unexported `hidden` is skipped), and the package-local Inner references are qualified
	// by the import alias.
	outer, ok := info.Structs["ext.Outer"]
	if !ok {
		t.Fatalf("expected ext.Outer to be enriched; have %v", structKeys(info.Structs))
	}
	wantOuter := []Field{
		{Name: "ID", Type: "string"},
		{Name: "Count", Type: "int"},
		{Name: "Inner", Type: "*ext.Inner"},
		{Name: "Tags", Type: "[]string"},
		{Name: "Items", Type: "[]*ext.Inner"},
		{Name: "ByName", Type: "map[string]*ext.Inner"},
	}
	if len(outer) != len(wantOuter) {
		t.Fatalf("ext.Outer fields = %d (%+v), want %d", len(outer), outer, len(wantOuter))
	}
	for i, w := range wantOuter {
		if outer[i].Name != w.Name || nospace(outer[i].Type) != nospace(w.Type) {
			t.Errorf("ext.Outer field %d = {%s %s}, want {%s %s}", i, outer[i].Name, outer[i].Type, w.Name, w.Type)
		}
	}

	inner, ok := info.Structs["ext.Inner"]
	if !ok {
		t.Fatalf("expected sibling ext.Inner to be enriched; have %v", structKeys(info.Structs))
	}
	if len(inner) != 1 || inner[0].Name != "Label" || inner[0].Type != "string" {
		t.Errorf("ext.Inner fields = %+v, want [{Label string}]", inner)
	}

	// ForeignMethods: the receiver method is recorded by qualified key with the `?`-relevant
	// facts (Arity, EndsInError) and Mode left at ModeNone.
	closeSig, ok := info.ForeignMethods["ext.Inner.Close"]
	if !ok {
		t.Fatalf("expected ext.Inner.Close in ForeignMethods; have %v", methodKeys(info.ForeignMethods))
	}
	if closeSig.Arity != 1 || !closeSig.EndsInError {
		t.Errorf("ext.Inner.Close = {Arity:%d EndsInError:%t}, want {1 true}", closeSig.Arity, closeSig.EndsInError)
	}
	if closeSig.Mode != ModeNone {
		t.Errorf("ext.Inner.Close.Mode = %v, want ModeNone", closeSig.Mode)
	}
}

// TestEnrichForeignResolveErrorIsNonFatal proves the error path: a resolver failure is
// collected and returned, never panics, and adds no foreign types (the import is simply
// left unknown).
func TestEnrichForeignResolveErrorIsNonFatal(t *testing.T) {
	src := `package p

import ext "example.com/ext"

derive func make(o *ext.Outer) int
`
	file, perr := parser.ParseFile(src)
	if perr != nil {
		t.Fatalf("parse: %v", perr)
	}
	info := Resolve(file)
	boom := func(string, string) (string, error) { return "", errResolve }
	errs := EnrichForeign(info, file.Imports, ".", boom)
	if len(errs) != 1 || errs[0] != errResolve {
		t.Fatalf("expected the resolver error returned non-fatally, got %v", errs)
	}
	if _, ok := info.Structs["ext.Outer"]; ok {
		t.Error("a failed resolve must add no foreign struct")
	}
}

// errResolve is a sentinel for the resolver-failure test.
var errResolve = errSentinel("resolve failed")

type errSentinel string

func (e errSentinel) Error() string { return string(e) }

// TestEnrichForeignNilResolverNoImportsIsSafe proves resolve==nil defaults to
// DefaultResolver without panicking; with no imports the resolver is never invoked, so
// the default path is exercised offline.
func TestEnrichForeignNilResolverNoImportsIsSafe(t *testing.T) {
	info := Resolve(nil)
	if errs := EnrichForeign(info, nil, ".", nil); errs != nil {
		t.Fatalf("expected no errors enriching an importless file, got %v", errs)
	}
}

func structKeys(m map[string][]Field) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func methodKeys(m map[string]FuncSig) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestEnrichForeignDrivenByASTImports double-checks the structural invariant via
// reflection over the input type: EnrichForeign consumes []*ast.ImportSpec from the
// parsed file, so the import list is the AST's, not a re-lexed one. (The no-scan.Lex
// guarantee is enforced statically by the package's dependency set; this asserts the
// import-spec shape the driver relies on.)
func TestEnrichForeignDrivenByASTImports(t *testing.T) {
	file, err := parser.ParseFile("package p\nimport ext \"example.com/ext\"\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(file.Imports) != 1 {
		t.Fatalf("expected 1 import spec, got %d", len(file.Imports))
	}
	if got := reflect.TypeOf(file.Imports).String(); got != "[]*ast.ImportSpec" {
		t.Errorf("file.Imports type = %s, want []*ast.ImportSpec", got)
	}
}
