package parser

import (
	"os"
	"path/filepath"
	"testing"

	"goal/internal/ast"
)

// readExample reads a feature example .goal input relative to the repo root
// (tests run with cwd = internal/parser).
func readExample(t *testing.T, rel string) string {
	t.Helper()
	path := filepath.Join("..", "..", rel)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

// findEnum returns the first EnumDecl in the file, or fails.
func findEnum(t *testing.T, f *ast.File, name string) *ast.EnumDecl {
	t.Helper()
	for _, d := range f.Decls {
		if e, ok := d.(*ast.EnumDecl); ok && e.Name != nil && e.Name.Name == name {
			return e
		}
	}
	t.Fatalf("enum %q not found", name)
	return nil
}

// findStruct returns the StructType declared as `type name struct ...`.
func findStruct(t *testing.T, f *ast.File, name string) *ast.StructType {
	t.Helper()
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, s := range gd.Specs {
			ts, ok := s.(*ast.TypeSpec)
			if !ok || ts.Name == nil || ts.Name.Name != name {
				continue
			}
			if st, ok := ts.Type.(*ast.StructType); ok {
				return st
			}
		}
	}
	t.Fatalf("struct %q not found", name)
	return nil
}

func TestParseEnumDecl(t *testing.T) {
	src := readExample(t, "features/01-enums/examples/status.goal")
	// The enum declaration is parsed before the function bodies, which use
	// labeled-argument construction (Status.Active(since: now())) — that is
	// US-022's grammar, not yet supported. The parser is error-tolerant, so the
	// EnumDecl is fully built regardless; this story asserts that declaration's
	// structure, so a body-level parse error here is expected and ignored.
	f, _ := ParseFile(src)

	e := findEnum(t, f, "Status")
	if len(e.Variants) != 3 {
		t.Fatalf("Status: want 3 variants, got %d", len(e.Variants))
	}

	// Pending: data-less.
	pending := e.Variants[0]
	if pending.Name.Name != "Pending" {
		t.Errorf("variant 0: want Pending, got %q", pending.Name.Name)
	}
	if pending.Payload != nil {
		t.Errorf("Pending should be data-less, got %d payload fields", len(pending.Payload))
	}

	// Active { since: Time }.
	active := e.Variants[1]
	if active.Name.Name != "Active" {
		t.Errorf("variant 1: want Active, got %q", active.Name.Name)
	}
	if len(active.Payload) != 1 {
		t.Fatalf("Active: want 1 payload field, got %d", len(active.Payload))
	}
	if got := active.Payload[0].Name.Name; got != "since" {
		t.Errorf("Active payload[0] name: want since, got %q", got)
	}
	if id, ok := active.Payload[0].Type.(*ast.Ident); !ok || id.Name != "Time" {
		t.Errorf("Active payload[0] type: want Ident Time, got %T %v", active.Payload[0].Type, active.Payload[0].Type)
	}

	// Cancelled { reason: string, at: Time }.
	cancelled := e.Variants[2]
	if cancelled.Name.Name != "Cancelled" {
		t.Errorf("variant 2: want Cancelled, got %q", cancelled.Name.Name)
	}
	if len(cancelled.Payload) != 2 {
		t.Fatalf("Cancelled: want 2 payload fields, got %d", len(cancelled.Payload))
	}
	wantFields := []struct{ name, typ string }{{"reason", "string"}, {"at", "Time"}}
	for i, w := range wantFields {
		pf := cancelled.Payload[i]
		if pf.Name.Name != w.name {
			t.Errorf("Cancelled payload[%d] name: want %s, got %q", i, w.name, pf.Name.Name)
		}
		if id, ok := pf.Type.(*ast.Ident); !ok || id.Name != w.typ {
			t.Errorf("Cancelled payload[%d] type: want Ident %s, got %T", i, w.typ, pf.Type)
		}
	}
}

func TestParseEnumDataless(t *testing.T) {
	src := readExample(t, "features/01-enums/examples/traffic.goal")
	f, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}
	e := findEnum(t, f, "Light")
	want := []string{"Red", "Yellow", "Green"}
	if len(e.Variants) != len(want) {
		t.Fatalf("Light: want %d variants, got %d", len(want), len(e.Variants))
	}
	for i, name := range want {
		v := e.Variants[i]
		if v.Name.Name != name {
			t.Errorf("variant %d: want %s, got %q", i, name, v.Name.Name)
		}
		if v.Payload != nil {
			t.Errorf("variant %s should be data-less", name)
		}
	}
}

func TestParseSealedInterface(t *testing.T) {
	src := readExample(t, "features/01-enums/examples/shape.goal")
	f, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}

	var sealed *ast.SealedInterfaceDecl
	for _, d := range f.Decls {
		if s, ok := d.(*ast.SealedInterfaceDecl); ok {
			sealed = s
			break
		}
	}
	if sealed == nil {
		t.Fatalf("no SealedInterfaceDecl found")
	}
	if sealed.Name == nil || sealed.Name.Name != "Shape" {
		t.Fatalf("sealed interface name: want Shape, got %v", sealed.Name)
	}
	if sealed.Methods == nil || len(sealed.Methods.List) != 0 {
		t.Errorf("Shape should have an empty method set, got %v", sealed.Methods)
	}

	// Circle and Rectangle declare `implements Shape`.
	for _, name := range []string{"Circle", "Rectangle"} {
		st := findStruct(t, f, name)
		if st.Implements == nil {
			t.Fatalf("%s should have an implements clause", name)
		}
		id, ok := st.Implements.Type.(*ast.Ident)
		if !ok || id.Name != "Shape" {
			t.Errorf("%s implements: want Ident Shape, got %T %v", name, st.Implements.Type, st.Implements.Type)
		}
	}
}

func TestParseImplements(t *testing.T) {
	cases := []struct {
		file       string
		structName string
		// want is the expected interface type: either a plain name (Ident) or
		// pkg.Name (SelectorExpr) when qualified is non-empty.
		name      string
		qualifier string
	}{
		{"features/07-implements/examples/value_recv.goal", "Point", "Stringer", ""},
		{"features/07-implements/examples/pointer_recv.goal", "Counter", "Resetter", ""},
		{"features/07-implements/examples/qualified_iface.goal", "Discard", "Writer", "io"},
	}
	for _, tc := range cases {
		t.Run(tc.structName, func(t *testing.T) {
			src := readExample(t, tc.file)
			f, err := ParseFile(src)
			if err != nil {
				t.Fatalf("ParseFile returned error: %v", err)
			}
			st := findStruct(t, f, tc.structName)
			if st.Implements == nil {
				t.Fatalf("%s should have an implements clause", tc.structName)
			}
			if tc.qualifier == "" {
				id, ok := st.Implements.Type.(*ast.Ident)
				if !ok || id.Name != tc.name {
					t.Fatalf("%s implements: want Ident %s, got %T %v", tc.structName, tc.name, st.Implements.Type, st.Implements.Type)
				}
				return
			}
			sel, ok := st.Implements.Type.(*ast.SelectorExpr)
			if !ok {
				t.Fatalf("%s implements: want SelectorExpr, got %T", tc.structName, st.Implements.Type)
			}
			x, ok := sel.X.(*ast.Ident)
			if !ok || x.Name != tc.qualifier {
				t.Errorf("%s implements qualifier: want %s, got %T %v", tc.structName, tc.qualifier, sel.X, sel.X)
			}
			if sel.Sel == nil || sel.Sel.Name != tc.name {
				t.Errorf("%s implements selector: want %s, got %v", tc.structName, tc.name, sel.Sel)
			}
		})
	}
}
