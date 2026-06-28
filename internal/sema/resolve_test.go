package sema

import (
	"strings"
	"testing"

	"goal/internal/parser"
)

// repr is a representative goal source exercising each construct sema resolves:
// an enum with a payload and a data-less variant, a struct, a plain function, the
// open-E Result / Option / closed-E Result signatures, a `from func` and a
// `derive func` conversion, value- and pointer-receiver methods, and a sealed
// interface.
const repr = `package demo

enum Status {
    Active { since: int }
    Pending
}

type Point struct {
	x int
	y int
}

sealed interface Shape {}

func area(p Point) float64 { return 0 }
func find() Result[int, error] { return Result.Ok(0) }
func lookup() Option[int] { return Option.None }
func closed() Result[int, MyErr] { return Result.Ok(0) }

from func parseInt(s string) int { return 0 }
derive func toPoint(p Pair) Point { return Point{} }

func (p Point) Dist() float64 { return 0 }
func (p *Point) Move(dx int) {}
`

// nospace strips whitespace so a type string is compared semantically (the AST
// printer canonicalizes spacing).
func nospace(s string) string { return strings.ReplaceAll(strings.Join(strings.Fields(s), ""), " ", "") }

func mustResolve(t *testing.T, src string) *Info {
	t.Helper()
	f, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return Resolve(f)
}

// TestResolveRepresentativeSource asserts sema's AST walk resolves the full set of
// name-keyed facts the transpiler depends on: enum variants and payloads, struct fields,
// sealed interfaces, function Result/Option signatures, the from/derive conversion
// registry, and per-receiver methods.
func TestResolveRepresentativeSource(t *testing.T) {
	s := mustResolve(t, repr)

	// Enums: variant names, ordering, and payload fields.
	se := s.Enums["Status"]
	if se == nil {
		t.Fatalf("Status enum missing")
	}
	if len(se.Variants) != 2 {
		t.Fatalf("Status variant count = %d, want 2", len(se.Variants))
	}
	if se.Variants[0].Name != "Active" || se.Variants[1].Name != "Pending" {
		t.Errorf("Status variants = %q/%q, want Active/Pending", se.Variants[0].Name, se.Variants[1].Name)
	}
	if len(se.Variants[0].Fields) != 1 || se.Variants[0].Fields[0].Name != "since" || nospace(se.Variants[0].Fields[0].Type) != "int" {
		t.Errorf("Active payload = %+v, want [{since int}]", se.Variants[0].Fields)
	}
	if len(se.Variants[1].Fields) != 0 {
		t.Errorf("Pending payload = %+v, want none", se.Variants[1].Fields)
	}
	if !se.VSet["Active"] || !se.VSet["Pending"] {
		t.Errorf("VSet incomplete: %v", se.VSet)
	}
	if !se.FieldSet["Active"]["since"] {
		t.Errorf("FieldSet missing Active.since: %v", se.FieldSet)
	}

	// Structs: ordered fields.
	ss := s.Structs["Point"]
	if len(ss) != 2 || ss[0].Name != "x" || ss[1].Name != "y" {
		t.Fatalf("Point fields = %+v, want [{x int} {y int}]", ss)
	}
	if nospace(ss[0].Type) != "int" || nospace(ss[1].Type) != "int" {
		t.Errorf("Point field types = %q/%q, want int/int", ss[0].Type, ss[1].Type)
	}

	// Sealed interface.
	if !s.Sealed["Shape"] {
		t.Errorf("Shape not recorded as sealed")
	}

	// Function signatures: mode, success/error types, arity, ends-in-error.
	if got := s.FuncSignatures["area"]; got.Mode != ModeNone {
		t.Errorf("area mode = %v, want ModeNone", got.Mode)
	}
	if got := s.FuncSignatures["find"]; got.Mode != ModeResult || got.T != "int" || got.E != "error" || got.Arity != 2 || !got.EndsInError {
		t.Errorf("find resolved wrong: %+v", got)
	}
	if got := s.FuncSignatures["lookup"]; got.Mode != ModeOption || got.T != "int" || got.Arity != 1 || got.EndsInError {
		t.Errorf("lookup resolved wrong: %+v", got)
	}
	if got := s.FuncSignatures["closed"]; got.Mode != ModeResultClosed || got.T != "int" || got.E != "MyErr" {
		t.Errorf("closed resolved wrong: %+v", got)
	}

	// From-registry: both the plain `from func` and the `derive func` are recorded under
	// their (src,dst) type keys with the conversion function name.
	if e, ok := s.FromRegistry[[2]string{"string", "int"}]; !ok || e.Name != "parseInt" {
		t.Errorf("from-registry [string,int] = %+v, ok=%v", e, ok)
	}
	if e, ok := s.FromRegistry[[2]string{"Pair", "Point"}]; !ok || e.Name != "toPoint" {
		t.Errorf("from-registry [Pair,Point] = %+v, ok=%v", e, ok)
	}

	// Methods: the per-receiver method-name set.
	if names := methodNames(s.Methods["Point"]); !names["Dist"] || !names["Move"] {
		t.Errorf("Point methods = %v, want Dist+Move", names)
	}
}

func methodNames(ms []Method) map[string]bool {
	out := map[string]bool{}
	for _, m := range ms {
		out[m.Name] = true
	}
	return out
}

// TestResolveStructCommaFieldType covers a struct field whose type contains an
// embedded (top-level) comma — here a generic instance Result[int, error]. The AST
// walk resolves it to a single, correctly-typed field (a whitespace/comma split would
// mangle it).
func TestResolveStructCommaFieldType(t *testing.T) {
	const src = `package demo

type Box struct {
	items Result[int, error]
	name  string
}
`
	s := mustResolve(t, src)
	fields := s.Structs["Box"]
	if len(fields) != 2 {
		t.Fatalf("Box fields = %d (%+v), want 2", len(fields), fields)
	}
	if fields[0].Name != "items" || nospace(fields[0].Type) != "Result[int,error]" {
		t.Errorf("field 0 = %+v, want {items Result[int, error]}", fields[0])
	}
	if fields[1].Name != "name" || fields[1].Type != "string" {
		t.Errorf("field 1 = %+v, want {name string}", fields[1])
	}
}
