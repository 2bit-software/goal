package sema

import (
	"path/filepath"
	"reflect"
	"testing"

	"goal/internal/analyze"
	"goal/internal/parser"
)

// TestEnrichForeignMatchesAnalyze is the US-001 parity gate: for the same multi-file
// input (a goal file importing a package with an exported struct AND an exported method),
// sema.EnrichForeign must produce ForeignMethods + Structs entries identical, field for
// field, to what analyze.EnrichForeign produces. The foreign package fixture is shared
// with analyze's own foreign tests (../analyze/testdata/extpkg), which declares both an
// exported struct (Outer/Inner, with package-local pointer/slice/map fields) and an
// exported method (Inner.Close), so the comparison exercises both maps.
func TestEnrichForeignMatchesAnalyze(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "analyze", "testdata", "extpkg"))
	if err != nil {
		t.Fatal(err)
	}
	// A `derive func` references ext.Outer, so analyze's needed-alias filter loads ext;
	// the goal AST's import list loads the same package for sema. Both then read the whole
	// foreign package, so the struct AND method sets must coincide.
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

	// Native (AST-driven) enrichment.
	file, perr := parser.ParseFile(src)
	if perr != nil {
		t.Fatalf("parse goal source: %v", perr)
	}
	info := Resolve(file)
	if errs := EnrichForeign(info, file.Imports, ".", resolve); len(errs) != 0 {
		t.Fatalf("sema.EnrichForeign errors: %v", errs)
	}

	// Legacy (token-scanning) enrichment, same input.
	tb := analyze.Build(src)
	if errs := analyze.EnrichForeign(tb, []string{src}, ".", resolve); len(errs) != 0 {
		t.Fatalf("analyze.EnrichForeign errors: %v", errs)
	}

	// Structs parity: every foreign struct analyze recorded must match sema's, field for
	// field (compared by Name+Type, since the two packages' Field types are distinct).
	for name, aFields := range tb.Structs {
		sFields, ok := info.Structs[name]
		if !ok {
			t.Errorf("sema.Structs missing %q (analyze has it)", name)
			continue
		}
		if !sameFields(sFields, aFields) {
			t.Errorf("Structs[%q] mismatch\n sema:    %+v\n analyze: %+v", name, sFields, aFields)
		}
	}
	if _, ok := info.Structs["ext.Outer"]; !ok {
		t.Fatalf("expected ext.Outer to be enriched; have %v", structKeys(info.Structs))
	}
	if _, ok := info.Structs["ext.Inner"]; !ok {
		t.Error("expected sibling ext.Inner to be enriched")
	}

	// ForeignMethods parity: every foreign method analyze recorded must match sema's by
	// the `?`-relevant facts (Arity, EndsInError); both leave Mode at ModeNone.
	for name, aSig := range tb.ForeignMethods {
		sSig, ok := info.ForeignMethods[name]
		if !ok {
			t.Errorf("sema.ForeignMethods missing %q (analyze has it)", name)
			continue
		}
		if sSig.Arity != aSig.Arity || sSig.EndsInError != aSig.EndsInError {
			t.Errorf("ForeignMethods[%q] mismatch\n sema:    {Arity:%d EndsInError:%t}\n analyze: {Arity:%d EndsInError:%t}",
				name, sSig.Arity, sSig.EndsInError, aSig.Arity, aSig.EndsInError)
		}
		if sSig.Mode != ModeNone {
			t.Errorf("ForeignMethods[%q].Mode = %v, want ModeNone", name, sSig.Mode)
		}
	}
	closeSig, ok := info.ForeignMethods["ext.Inner.Close"]
	if !ok {
		t.Fatalf("expected ext.Inner.Close in ForeignMethods; have %v", methodKeys(info.ForeignMethods))
	}
	if closeSig.Arity != 1 || !closeSig.EndsInError {
		t.Errorf("ext.Inner.Close = {Arity:%d EndsInError:%t}, want {1 true}", closeSig.Arity, closeSig.EndsInError)
	}
}

// TestEnrichForeignResolveErrorIsNonFatal proves the error path: a resolver failure is
// collected and returned, never panics, and adds no foreign types (the import is simply
// left unknown, matching analyze).
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

func sameFields(s []Field, a []analyze.Field) bool {
	if len(s) != len(a) {
		return false
	}
	for i := range s {
		if s[i].Name != a[i].Name || s[i].Type != a[i].Type {
			return false
		}
	}
	return true
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
