package interp

// These tests prove US-013 "Eval statement-position match": the interpreter
// dispatches a statement-position `match` on the scrutinee variant's tag, binds
// the matched payload into scope, and runs the selected arm. They drive a real
// parsed+resolved 02-match-shaped program through the direct-evalExpr seam
// (newInterp + call, from call_test.go), asserting that the correct arm runs for
// each variant — including the data-less and `_` rest forms — and that the
// defensive default of a proven-exhaustive match is a loud `unreachable` panic,
// never a silent fall-through.

import (
	"errors"
	"strings"
	"testing"

	"goal/internal/ast"
)

// matchProgram is a 02-match-shaped program: an enum with a data-less variant
// and two payload-carrying variants, a function that matches and returns an
// observable int per arm (binding and reading the payload), and a function with
// a `_` rest arm. Wrapper functions construct each variant and drive the match.
const matchProgram = `package main

enum Shape {
	Point
	Circle { radius: int }
	Square { side: int }
}

func area(s Shape) int {
	match s {
		Shape.Point => return 0
		Shape.Circle(c) => return c.radius * c.radius
		Shape.Square(q) => return q.side * q.side
	}
	return -1
}

func describe(s Shape) int {
	match s {
		Shape.Circle(c) => return c.radius
		_ => return 0
	}
	return -1
}

func pointArea() int  { return area(Shape.Point) }
func circleArea() int { return area(Shape.Circle(radius: 3)) }
func squareArea() int { return area(Shape.Square(side: 4)) }

func describeCircle() int { return describe(Shape.Circle(radius: 7)) }
func describeSquare() int { return describe(Shape.Square(side: 9)) }
`

func TestMatchDispatchesDataLessArm(t *testing.T) {
	ip := newInterp(t, matchProgram)
	if got := evalFn(t, ip, "pointArea"); !got.Equal(IntVal(0)) {
		t.Errorf("pointArea() = %s, want 0", got)
	}
}

func TestMatchDispatchesPayloadArmAndBindsPayload(t *testing.T) {
	ip := newInterp(t, matchProgram)
	if got := evalFn(t, ip, "circleArea"); !got.Equal(IntVal(9)) {
		t.Errorf("circleArea() = %s, want 9 (c.radius*c.radius for radius 3)", got)
	}
	if got := evalFn(t, ip, "squareArea"); !got.Equal(IntVal(16)) {
		t.Errorf("squareArea() = %s, want 16 (q.side*q.side for side 4)", got)
	}
}

func TestMatchRestArmRunsWhenNoVariantMatches(t *testing.T) {
	ip := newInterp(t, matchProgram)
	// Circle is named explicitly -> its arm runs and binds the payload.
	if got := evalFn(t, ip, "describeCircle"); !got.Equal(IntVal(7)) {
		t.Errorf("describeCircle() = %s, want 7", got)
	}
	// Square has no explicit arm -> the `_` rest arm runs.
	if got := evalFn(t, ip, "describeSquare"); !got.Equal(IntVal(0)) {
		t.Errorf("describeSquare() = %s, want 0 (rest arm)", got)
	}
}

// TestMatchDefaultArmIsUnreachable proves the defensive default of a
// proven-exhaustive match is a loud panic, never a silent fall-through. Because
// a sema-checked program cannot construct a variant whose tag escapes the arms,
// the scenario is built directly: a match with arms for only Circle/Square is
// run against a Point value, so no arm and no rest covers the tag.
func TestMatchDefaultArmIsUnreachable(t *testing.T) {
	ip := newInterp(t, matchProgram)
	scope := ip.root.NewChild()
	scope.Define("s", VariantVal("Shape", "Point", nil))

	m := &ast.MatchExpr{
		Subject: &ast.Ident{Name: "s"},
		Arms: []*ast.MatchArm{
			{
				Pattern: &ast.VariantPattern{Enum: &ast.Ident{Name: "Shape"}, Variant: &ast.Ident{Name: "Circle"}},
				Body:    &ast.ReturnStmt{Results: []ast.Expr{intLit("1")}},
			},
			{
				Pattern: &ast.VariantPattern{Enum: &ast.Ident{Name: "Shape"}, Variant: &ast.Ident{Name: "Square"}},
				Body:    &ast.ReturnStmt{Results: []ast.Expr{intLit("2")}},
			},
		},
	}

	err := ip.execStmt(&ast.ExprStmt{X: m}, scope)
	if err == nil {
		t.Fatalf("matching Point against {Circle,Square}: expected an unreachable panic, got none")
	}
	var ps panicSignal
	if !errors.As(err, &ps) {
		t.Fatalf("error = %v, want a panicSignal", err)
	}
	if !strings.Contains(ps.value.Str, "unreachable") {
		t.Errorf("panic value = %q, want it to mention `unreachable`", ps.value.Str)
	}
}

// TestMatchOnNonVariantIsRefused proves a match scrutinee that is not a variant
// is a descriptive refusal, never a silent no-op.
func TestMatchOnNonVariantIsRefused(t *testing.T) {
	ip := newInterp(t, matchProgram)
	scope := ip.root.NewChild()
	scope.Define("n", IntVal(5))

	m := &ast.MatchExpr{
		Subject: &ast.Ident{Name: "n"},
		Arms: []*ast.MatchArm{
			{Pattern: &ast.RestPattern{}, Body: &ast.ReturnStmt{}},
		},
	}
	err := ip.execStmt(&ast.ExprStmt{X: m}, scope)
	if err == nil {
		t.Fatalf("matching a non-variant: expected an error, got none")
	}
	if !strings.Contains(err.Error(), "variant") {
		t.Errorf("error %q does not explain the subject must be a variant", err)
	}
}
