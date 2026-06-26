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
