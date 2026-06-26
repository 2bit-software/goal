package pass

import (
	"go/format"
	"strings"
	"testing"

	"goal/internal/analyze"
)

// injectForeign mirrors what analyze.EnrichForeign does for an imported package, so the
// lowering can be tested without touching the disk: it adds qualified struct field sets
// to the tables built from src.
func injectForeign(t *analyze.Tables, structs map[string][]analyze.Field) {
	for name, fields := range structs {
		t.Structs[name] = fields
		t.TypeDecls[name] = "struct"
	}
}

func TestDeriveForeignPointerSource(t *testing.T) {
	src := `package conv

import ext "example.com/ext"

type Local struct {
	ID    string
	Count int
	Tags  []string
}

derive func make(o *ext.Outer) Local
`
	tables := analyze.Build(src)
	injectForeign(tables, map[string][]analyze.Field{
		"ext.Outer": {
			{Name: "ID", Type: "string"},
			{Name: "Count", Type: "int"},
			{Name: "Tags", Type: "[]string"},
			{Name: "Inner", Type: "*ext.Inner"}, // extra foreign field Local does not source
		},
	})
	got := mustDerive(t, src, tables)
	for _, want := range []string{
		"func make(o *ext.Outer) Local",
		"out.ID = o.ID",
		"out.Count = o.Count",
		"out.Tags = o.Tags",
		"return out",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated Go missing %q\n--- got ---\n%s", want, got)
		}
	}
}

func TestDeriveForeignPointerTarget(t *testing.T) {
	src := `package conv

import ext "example.com/ext"

type Local struct {
	ID string
}

derive func up(l Local) *ext.Small
`
	tables := analyze.Build(src)
	injectForeign(tables, map[string][]analyze.Field{
		"ext.Small": {{Name: "ID", Type: "string"}},
	})
	got := mustDerive(t, src, tables)
	for _, want := range []string{
		"func up(l Local) *ext.Small",
		"var out ext.Small", // built as a value, not a nil pointer
		"out.ID = l.ID",
		"return &out", // returned by address
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated Go missing %q\n--- got ---\n%s", want, got)
		}
	}
}

// TestDeriveForeignPointerSourceNilGuard pins the nil guard a pointer source earns: a nil
// argument (e.g. an absent proto sub-message) derives to the zero target instead of panicking
// on field access.
func TestDeriveForeignPointerSourceNilGuard(t *testing.T) {
	src := `package conv

import ext "example.com/ext"

type Local struct {
	ID string
}

derive func make(o *ext.Outer) Local
`
	tables := analyze.Build(src)
	injectForeign(tables, map[string][]analyze.Field{
		"ext.Outer": {{Name: "ID", Type: "string"}},
	})
	got := mustDerive(t, src, tables)
	for _, want := range []string{
		"if o == nil {",
		"return out", // the zero Local, not a deref panic
		"out.ID = o.ID",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated Go missing %q\n--- got ---\n%s", want, got)
		}
	}
}

// TestDeriveFuncIsRegistryLeaf pins that a `derive func` registers as a conversion leaf, so one
// derived conversion can source a field of another — here the element conversion of a slice the
// outer deriver recurses into. Without registration this fails as an unbridged element.
func TestDeriveFuncIsRegistryLeaf(t *testing.T) {
	src := `package conv

import ext "example.com/ext"

type Elem struct {
	ID string
}

type Bag struct {
	Items []Elem
}

derive func elem(e *ext.E) Elem
derive func bag(b *ext.B) Bag
`
	tables := analyze.Build(src)
	injectForeign(tables, map[string][]analyze.Field{
		"ext.E": {{Name: "ID", Type: "string"}},
		"ext.B": {{Name: "Items", Type: "[]*ext.E"}},
	})
	got := mustDerive(t, src, tables)
	for _, want := range []string{
		"func elem(e *ext.E) Elem",
		"out.Items = make([]Elem, len(b.Items))",
		"out.Items[i] = elem(b.Items[i])", // slice recursion resolves the element via the derived leaf
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated Go missing %q\n--- got ---\n%s", want, got)
		}
	}
}

func mustDerive(t *testing.T, src string, tables *analyze.Tables) string {
	t.Helper()
	out, err := Derive(src, tables)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	formatted, err := format.Source([]byte(out))
	if err != nil {
		t.Fatalf("generated Go did not format: %v\n--- src ---\n%s", err, out)
	}
	return string(formatted)
}
