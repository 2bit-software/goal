package analyze

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseImports(t *testing.T) {
	src := `package domain

import (
	hobv1 "github.com/2bit-software/hob/gen/hob/v1"
	"fmt"
	_ "embed"
	. "errors"
)

import "strings"
`
	got := ParseImports(src)
	want := []ImportSpec{
		{Alias: "hobv1", Path: "github.com/2bit-software/hob/gen/hob/v1"},
		{Path: "fmt"},
		{Alias: "_", Path: "embed"},
		{Alias: ".", Path: "errors"},
		{Path: "strings"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseImports mismatch\n got: %+v\nwant: %+v", got, want)
	}
}

func TestNeededAliasesOnlyDeriveAndFrom(t *testing.T) {
	src := `package p

import ext "example.com/ext"

func unrelated(x ext.Thing) {}        // not derive/from — must NOT pull ext in

from func conv(s pb.Raw) Local
derive func mk(o *ext.Outer) Local
`
	got := neededAliases([]string{src})
	if !got["ext"] || !got["pb"] {
		t.Errorf("expected ext and pb as needed aliases, got %v", got)
	}
	// A qualifier used only outside derive/from (the unrelated func param) is still ext,
	// which is legitimately needed here; assert the set is exactly {ext, pb}.
	if len(got) != 2 {
		t.Errorf("expected exactly {ext, pb}, got %v", got)
	}
}

func TestEnrichForeignExtractsExportedStructs(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("testdata", "extpkg"))
	if err != nil {
		t.Fatal(err)
	}
	src := `package consumer

import ext "example.com/ext"

type Local struct{ id string }

derive func make(o *ext.Outer) Local
`
	// Inject a resolver so the test needs no go toolchain: map the import path to the
	// on-disk fixture directory.
	resolve := func(importPath, fromDir string) (string, error) {
		if importPath != "example.com/ext" {
			t.Fatalf("unexpected import path %q", importPath)
		}
		return dir, nil
	}
	tb := newTables()
	if errs := EnrichForeign(tb, []string{src}, ".", resolve); len(errs) != 0 {
		t.Fatalf("EnrichForeign errors: %v", errs)
	}

	outer, ok := tb.Structs["ext.Outer"]
	if !ok {
		t.Fatalf("ext.Outer not extracted; have %v", keys(tb.Structs))
	}
	want := []Field{
		{Name: "ID", Type: "string"},
		{Name: "Count", Type: "int"},
		{Name: "Inner", Type: "*ext.Inner"},
		{Name: "Tags", Type: "[]string"},
		{Name: "Items", Type: "[]*ext.Inner"},
		{Name: "ByName", Type: "map[string]*ext.Inner"},
	}
	if !reflect.DeepEqual(outer, want) {
		t.Errorf("ext.Outer fields mismatch\n got: %+v\nwant: %+v", outer, want)
	}
	if _, ok := tb.Structs["ext.Inner"]; !ok {
		t.Error("ext.Inner (a sibling exported struct) should also be extracted")
	}
	if _, ok := tb.Structs["ext.unexported"]; ok {
		t.Error("unexported type must not be extracted")
	}
	if _, ok := tb.TypeDecls["ext.Outer"]; !ok {
		t.Error(`ext.Outer should be recorded in TypeDecls as "struct"`)
	}
}

func TestEnrichForeignSkipsUnreferencedImports(t *testing.T) {
	// No derive/from at all → no foreign loading, even with an import present.
	src := `package p
import ext "example.com/ext"
func f() {}
`
	called := false
	resolve := func(string, string) (string, error) { called = true; return "", nil }
	tb := newTables()
	EnrichForeign(tb, []string{src}, ".", resolve)
	if called {
		t.Error("resolver should not be called when no derive/from references a qualifier")
	}
}

func keys(m map[string][]Field) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
