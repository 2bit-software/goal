package sema

import (
	"strings"
	"testing"

	"goal/internal/analyze"
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

// nospace strips whitespace so a type string is compared semantically (the token
// scanner captures verbatim source; the AST printer canonicalizes spacing).
func nospace(s string) string { return strings.ReplaceAll(strings.Join(strings.Fields(s), ""), " ", "") }

func mustResolve(t *testing.T, src string) *Info {
	t.Helper()
	f, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return Resolve(f)
}

// TestResolveMatchesAnalyze asserts sema (AST walk) resolves the same name-keyed
// symbols as analyze.Build (token scan) for the representative source.
func TestResolveMatchesAnalyze(t *testing.T) {
	a := analyze.Build(repr)
	s := mustResolve(t, repr)

	// Enums: variant names and payload fields agree.
	ae, se := a.Enums["Status"], s.Enums["Status"]
	if ae == nil || se == nil {
		t.Fatalf("Status enum missing: analyze=%v sema=%v", ae, se)
	}
	if len(ae.Variants) != len(se.Variants) {
		t.Fatalf("Status variant count: analyze=%d sema=%d", len(ae.Variants), len(se.Variants))
	}
	for i := range ae.Variants {
		av, sv := ae.Variants[i], se.Variants[i]
		if av.Name != sv.Name {
			t.Errorf("variant %d name: analyze=%q sema=%q", i, av.Name, sv.Name)
		}
		if len(av.Fields) != len(sv.Fields) {
			t.Fatalf("variant %s field count: analyze=%d sema=%d", av.Name, len(av.Fields), len(sv.Fields))
		}
		for j := range av.Fields {
			if av.Fields[j].Name != sv.Fields[j].Name || nospace(av.Fields[j].Type) != nospace(sv.Fields[j].Type) {
				t.Errorf("variant %s field %d: analyze=%+v sema=%+v", av.Name, j, av.Fields[j], sv.Fields[j])
			}
		}
	}
	if !se.VSet["Active"] || !se.VSet["Pending"] {
		t.Errorf("sema VSet incomplete: %v", se.VSet)
	}
	if !se.FieldSet["Active"]["since"] {
		t.Errorf("sema FieldSet missing Active.since: %v", se.FieldSet)
	}

	// Structs: ordered fields agree.
	as, ss := a.Structs["Point"], s.Structs["Point"]
	if len(as) != len(ss) {
		t.Fatalf("Point field count: analyze=%d sema=%d", len(as), len(ss))
	}
	for i := range as {
		if as[i].Name != ss[i].Name || nospace(as[i].Type) != nospace(ss[i].Type) {
			t.Errorf("Point field %d: analyze=%+v sema=%+v", i, as[i], ss[i])
		}
	}

	// Sealed interface.
	if !a.Sealed["Shape"] || !s.Sealed["Shape"] {
		t.Errorf("Shape sealed: analyze=%v sema=%v", a.Sealed["Shape"], s.Sealed["Shape"])
	}

	// Function signatures: mode, success/error types, arity, ends-in-error.
	for _, name := range []string{"area", "find", "lookup", "closed"} {
		af, sf := a.FuncSignatures[name], s.FuncSignatures[name]
		if int(af.Mode) != int(sf.Mode) {
			t.Errorf("%s mode: analyze=%d sema=%d", name, af.Mode, sf.Mode)
		}
		if nospace(af.T) != nospace(sf.T) || nospace(af.E) != nospace(sf.E) {
			t.Errorf("%s T/E: analyze=(%q,%q) sema=(%q,%q)", name, af.T, af.E, sf.T, sf.E)
		}
		if af.Arity != sf.Arity || af.EndsInError != sf.EndsInError {
			t.Errorf("%s arity/endsErr: analyze=(%d,%v) sema=(%d,%v)", name, af.Arity, af.EndsInError, sf.Arity, sf.EndsInError)
		}
	}
	// Spot-check the resolved Result/Option facts directly.
	if got := s.FuncSignatures["find"]; got.Mode != ModeResult || got.T != "int" || got.E != "error" || got.Arity != 2 || !got.EndsInError {
		t.Errorf("find resolved wrong: %+v", got)
	}
	if got := s.FuncSignatures["lookup"]; got.Mode != ModeOption || got.T != "int" || got.Arity != 1 || got.EndsInError {
		t.Errorf("lookup resolved wrong: %+v", got)
	}
	if got := s.FuncSignatures["closed"]; got.Mode != ModeResultClosed || got.T != "int" || got.E != "MyErr" {
		t.Errorf("closed resolved wrong: %+v", got)
	}

	// From-registry: every analyze conversion entry is present in sema with the
	// same name and fallibility (compared modulo whitespace in the key types).
	for k, av := range a.FromRegistry {
		found := false
		for sk, sv := range s.FromRegistry {
			if nospace(sk[0]) == nospace(k[0]) && nospace(sk[1]) == nospace(k[1]) {
				found = true
				if sv.Name != av.Name || sv.Fallible != av.Fallible {
					t.Errorf("from-registry %v: analyze=%+v sema=%+v", k, av, sv)
				}
			}
		}
		if !found {
			t.Errorf("sema from-registry missing analyze key %v (%+v)", k, av)
		}
	}
	if e, ok := s.FromRegistry[[2]string{"string", "int"}]; !ok || e.Name != "parseInt" {
		t.Errorf("sema from-registry [string,int] = %+v, ok=%v", e, ok)
	}
	if e, ok := s.FromRegistry[[2]string{"Pair", "Point"}]; !ok || e.Name != "toPoint" {
		t.Errorf("sema from-registry [Pair,Point] = %+v, ok=%v", e, ok)
	}

	// Methods: the per-receiver method-name set agrees.
	if names := methodNames(s.Methods["Point"]); !names["Dist"] || !names["Move"] {
		t.Errorf("sema Point methods = %v, want Dist+Move", names)
	}
	if names := methodNames(a.Methods["Point"]); !names["Dist"] || !names["Move"] {
		t.Errorf("analyze Point methods = %v, want Dist+Move (sanity)", names)
	}
}

func methodNames[M any](ms []M) map[string]bool {
	out := map[string]bool{}
	for _, m := range ms {
		switch v := any(m).(type) {
		case Method:
			out[v.Name] = true
		case analyze.Method:
			out[v.Name] = true
		}
	}
	return out
}

// TestResolveStructCommaFieldType is the analyze comma-split-bug case: a struct
// field whose type contains an embedded (top-level) comma — here a generic
// instance Result[int, error]. The token scanner splits the field text on
// whitespace and mangles the comma-bearing type; the AST walk resolves it to a
// single, correctly-typed field.
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

	// Demonstrate the bug the AST walk fixes: the token scanner's whitespace split
	// does NOT produce the same clean two-field result for the comma-bearing type.
	a := analyze.Build(src)
	if got := a.Structs["Box"]; len(got) == 2 &&
		got[0].Name == "items" && nospace(got[0].Type) == "Result[int,error]" &&
		got[1].Name == "name" && got[1].Type == "string" {
		t.Errorf("expected analyze to mishandle the comma-bearing field, but it resolved cleanly: %+v", got)
	}
}
