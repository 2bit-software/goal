package interp

// These tests prove US-014 "Eval value-position match": the interpreter
// evaluates a `match` used as an expression — `return match`, `x := match`, and
// `var x = match` — yielding the selected arm's value as an expression result.
// They drive a real parsed+resolved 02-match-shaped program through the
// direct-evalExpr seam (newInterp + evalFn, from call_test.go/composite_test.go),
// asserting the right value flows out of each value position for every input
// variant, including a payload-computing arm and a `_` rest arm, and that the
// defensive default of a proven-exhaustive match is a loud `unreachable` panic.

import (
	"errors"
	"strings"
	"testing"

	"goal/internal/ast"
)

// valueMatchProgram is a 02-match-shaped program whose match sits in VALUE
// position in all three forms: `return match` (areaReturn), `x := match`
// (areaShortVar), and `var x = match` (areaVarDecl). A payload arm computes its
// value from the bound payload (Circle => c.radius*c.radius). descByVal uses a
// `_` rest arm to supply a value when no variant arm matches. Wrapper functions
// construct each variant and drive the match.
const valueMatchProgram = `package main

enum Shape {
	Point
	Circle { radius: int }
	Square { side: int }
}

func areaReturn(s Shape) int {
	return match s {
		Shape.Point => 0
		Shape.Circle(c) => c.radius * c.radius
		Shape.Square(q) => q.side * q.side
	}
}

func areaShortVar(s Shape) int {
	a := match s {
		Shape.Point => 0
		Shape.Circle(c) => c.radius * c.radius
		Shape.Square(q) => q.side * q.side
	}
	return a
}

func areaVarDecl(s Shape) int {
	var a = match s {
		Shape.Point => 0
		Shape.Circle(c) => c.radius * c.radius
		Shape.Square(q) => q.side * q.side
	}
	return a
}

func descByVal(s Shape) int {
	return match s {
		Shape.Circle(c) => c.radius
		_ => 0
	}
}

func retPoint() int  { return areaReturn(Shape.Point) }
func retCircle() int { return areaReturn(Shape.Circle(radius: 3)) }
func retSquare() int { return areaReturn(Shape.Square(side: 4)) }

func shortPoint() int  { return areaShortVar(Shape.Point) }
func shortCircle() int { return areaShortVar(Shape.Circle(radius: 3)) }
func shortSquare() int { return areaShortVar(Shape.Square(side: 4)) }

func varPoint() int  { return areaVarDecl(Shape.Point) }
func varCircle() int { return areaVarDecl(Shape.Circle(radius: 3)) }
func varSquare() int { return areaVarDecl(Shape.Square(side: 4)) }

func descCircle() int { return descByVal(Shape.Circle(radius: 7)) }
func descSquare() int { return descByVal(Shape.Square(side: 9)) }
`

// TestValueMatchInReturn proves `return match` yields the selected arm's value
// for each input variant (data-less, and two payload arms that compute from the
// bound payload).
func TestValueMatchInReturn(t *testing.T) {
	ip := newInterp(t, valueMatchProgram)
	cases := []struct {
		fn   string
		want int64
	}{
		{"retPoint", 0},
		{"retCircle", 9}, // c.radius*c.radius for radius 3
		{"retSquare", 16}, // q.side*q.side for side 4
	}
	for _, c := range cases {
		if got := evalFn(t, ip, c.fn); !got.Equal(IntVal(c.want)) {
			t.Errorf("%s() = %s, want %d", c.fn, got, c.want)
		}
	}
}

// TestValueMatchInShortVar proves `x := match` binds the selected arm's value.
func TestValueMatchInShortVar(t *testing.T) {
	ip := newInterp(t, valueMatchProgram)
	cases := []struct {
		fn   string
		want int64
	}{
		{"shortPoint", 0},
		{"shortCircle", 9},
		{"shortSquare", 16},
	}
	for _, c := range cases {
		if got := evalFn(t, ip, c.fn); !got.Equal(IntVal(c.want)) {
			t.Errorf("%s() = %s, want %d", c.fn, got, c.want)
		}
	}
}

// TestValueMatchInVarDecl proves `var x = match` binds the selected arm's value.
func TestValueMatchInVarDecl(t *testing.T) {
	ip := newInterp(t, valueMatchProgram)
	cases := []struct {
		fn   string
		want int64
	}{
		{"varPoint", 0},
		{"varCircle", 9},
		{"varSquare", 16},
	}
	for _, c := range cases {
		if got := evalFn(t, ip, c.fn); !got.Equal(IntVal(c.want)) {
			t.Errorf("%s() = %s, want %d", c.fn, got, c.want)
		}
	}
}

// TestValueMatchRestArmSuppliesValue proves a `_` rest arm yields the value when
// no variant arm matches, while a named variant arm still binds and reads its
// payload.
func TestValueMatchRestArmSuppliesValue(t *testing.T) {
	ip := newInterp(t, valueMatchProgram)
	if got := evalFn(t, ip, "descCircle"); !got.Equal(IntVal(7)) {
		t.Errorf("descCircle() = %s, want 7 (Circle arm binds radius)", got)
	}
	if got := evalFn(t, ip, "descSquare"); !got.Equal(IntVal(0)) {
		t.Errorf("descSquare() = %s, want 0 (rest arm)", got)
	}
}

// TestValueMatchDefaultIsUnreachable proves the defensive default of a value-
// position match is a loud panic, never a silent zero value. A sema-checked
// program cannot construct a variant whose tag escapes the arms, so the scenario
// is built directly: a value-position match with arms for only Circle/Square is
// evaluated against a Point value, so no arm and no rest covers the tag.
func TestValueMatchDefaultIsUnreachable(t *testing.T) {
	ip := newInterp(t, valueMatchProgram)
	scope := ip.root.NewChild()
	scope.Define("s", VariantVal("Shape", "Point", nil))

	m := &ast.MatchExpr{
		Subject: &ast.Ident{Name: "s"},
		Arms: []*ast.MatchArm{
			{
				Pattern: &ast.VariantPattern{Enum: &ast.Ident{Name: "Shape"}, Variant: &ast.Ident{Name: "Circle"}},
				Body:    intLit("1"),
			},
			{
				Pattern: &ast.VariantPattern{Enum: &ast.Ident{Name: "Shape"}, Variant: &ast.Ident{Name: "Square"}},
				Body:    intLit("2"),
			},
		},
	}

	_, err := ip.evalExpr(m, scope)
	if err == nil {
		t.Fatalf("value-match Point against {Circle,Square}: expected an unreachable panic, got none")
	}
	var ps panicSignal
	if !errors.As(err, &ps) {
		t.Fatalf("error = %v, want a panicSignal", err)
	}
	if !strings.Contains(ps.value.Str, "unreachable") {
		t.Errorf("panic value = %q, want it to mention `unreachable`", ps.value.Str)
	}
}

// TestValueMatchOnNonVariantIsRefused proves a value-position match scrutinee
// that is not a variant is a descriptive refusal, never a silent zero value.
func TestValueMatchOnNonVariantIsRefused(t *testing.T) {
	ip := newInterp(t, valueMatchProgram)
	scope := ip.root.NewChild()
	scope.Define("n", IntVal(5))

	m := &ast.MatchExpr{
		Subject: &ast.Ident{Name: "n"},
		Arms: []*ast.MatchArm{
			{Pattern: &ast.RestPattern{}, Body: intLit("0")},
		},
	}
	_, err := ip.evalExpr(m, scope)
	if err == nil {
		t.Fatalf("value-match on a non-variant: expected an error, got none")
	}
	if !strings.Contains(err.Error(), "variant") {
		t.Errorf("error %q does not explain the subject must be a variant", err)
	}
}
