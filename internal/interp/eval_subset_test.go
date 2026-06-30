package interp

// US-012 behavioral gate: a subset of the goscript evaluation conformance suite
// that exercises the ported expression EVALUATOR (eval.goal) directly, WITHOUT
// the US-013 driver (New / findMain / Run / registration). It constructs a bare
// *Interp and a fresh scope and runs parsed expressions through the real
// evalExpr — the same arithmetic, comparison, logical (short-circuiting), unary,
// and error matrix as eval_test.go, but reachable at the US-012 checkpoint where
// only value/env (US-011) and the evaluator (US-012) are ported.
//
// This file is the AC-2 parity oracle for US-012: the internal/selfhost port
// harness copies it beside the transpiled internal/compiler/interp package and
// runs it against the goal-built evaluator, while `task check` runs the same
// file against the legacy package — same input, same output proves the
// goal-sourced evaluator is correct.

import (
	"strings"
	"testing"

	"goal/internal/ast"
	"goal/internal/parser"
)

// subsetExprOf parses `package main\nfunc main() { <expr> }`, returns a
// driver-free *Interp, a fresh evaluation scope, and the parsed expression. It
// finds main's first expression statement by walking the declarations directly
// (not via the US-013 findMain), so it depends on nothing beyond the parser and
// the ported evaluator.
func subsetExprOf(t *testing.T, expr string) (*Interp, *Env, ast.Expr) {
	t.Helper()
	src := "package main\nfunc main() {\n" + expr + "\n}\n"
	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile(%q): %v", expr, err)
	}
	var es *ast.ExprStmt
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name == nil || fn.Name.Name != "main" {
			continue
		}
		if fn.Body == nil || len(fn.Body.List) == 0 {
			t.Fatalf("expr %q: no statement in main", expr)
		}
		s, ok := fn.Body.List[0].(*ast.ExprStmt)
		if !ok {
			t.Fatalf("expr %q: first statement is %T, want *ast.ExprStmt", expr, fn.Body.List[0])
		}
		es = s
	}
	if es == nil {
		t.Fatalf("expr %q: func main not found", expr)
	}
	return &Interp{}, NewEnv(), es.X
}

// subsetEval evaluates a single expression program through the real evalExpr,
// failing on any evaluation error.
func subsetEval(t *testing.T, expr string) Value {
	t.Helper()
	ip, scope, x := subsetExprOf(t, expr)
	v, err := ip.evalExpr(x, scope)
	if err != nil {
		t.Fatalf("evalExpr(%q): unexpected error: %v", expr, err)
	}
	return v
}

func TestEvalSubsetExpressions(t *testing.T) {
	cases := []struct {
		name string
		expr string
		want Value
	}{
		{"int literal", "42", IntVal(42)},
		{"hex int literal", "0x10", IntVal(16)},
		{"float literal", "3.5", FloatVal(3.5)},
		{"string literal", `"hello"`, StrVal("hello")},
		{"true literal", "true", BoolVal(true)},
		{"false literal", "false", BoolVal(false)},
		{"int add", "2 + 3", IntVal(5)},
		{"int precedence", "2 + 3 * 4", IntVal(14)},
		{"paren regroup", "(2 + 3) * 4", IntVal(20)},
		{"int div truncates", "7 / 2", IntVal(3)},
		{"int rem", "7 % 3", IntVal(1)},
		{"float div", "7.0 / 2.0", FloatVal(3.5)},
		{"string concat", `"foo" + "bar"`, StrVal("foobar")},
		{"int less", "2 < 3", BoolVal(true)},
		{"int geq false", "2 >= 3", BoolVal(false)},
		{"int equal", "3 == 3", BoolVal(true)},
		{"int not equal", "3 != 4", BoolVal(true)},
		{"string less", `"a" < "b"`, BoolVal(true)},
		{"float greater", "3.5 > 2.5", BoolVal(true)},
		{"and true", "true && true", BoolVal(true)},
		{"and false", "true && false", BoolVal(false)},
		{"or true", "false || true", BoolVal(true)},
		{"or false", "false || false", BoolVal(false)},
		{"unary neg int", "-5", IntVal(-5)},
		{"unary neg float", "-2.5", FloatVal(-2.5)},
		{"unary not", "!false", BoolVal(true)},
		{"combined logical compare", "(2 < 3) && (4 > 1)", BoolVal(true)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := subsetEval(t, c.expr)
			if !got.Equal(c.want) {
				t.Fatalf("eval %q = %s, want %s", c.expr, got.String(), c.want.String())
			}
		})
	}
}

// TestEvalSubsetShortCircuit proves the right operand is not evaluated when the
// left already decides the result: a divide-by-zero on the right would error if
// evaluated, but these evaluate cleanly.
func TestEvalSubsetShortCircuit(t *testing.T) {
	if got := subsetEval(t, "false && (1 / 0 == 0)"); !got.Equal(BoolVal(false)) {
		t.Fatalf("false && ... = %s, want false (right side must not be evaluated)", got.String())
	}
	if got := subsetEval(t, "true || (1 / 0 == 0)"); !got.Equal(BoolVal(true)) {
		t.Fatalf("true || ... = %s, want true (right side must not be evaluated)", got.String())
	}
}

// TestEvalSubsetShortCircuitEvaluatesRight confirms the right operand IS
// evaluated when the left does not decide the result, so the error surfaces.
func TestEvalSubsetShortCircuitEvaluatesRight(t *testing.T) {
	ip, scope, x := subsetExprOf(t, "true && (1 / 0 == 0)")
	if _, err := ip.evalExpr(x, scope); err == nil {
		t.Fatal("true && (1/0==0): want divide-by-zero error, got nil")
	}
}

func TestEvalSubsetErrors(t *testing.T) {
	cases := []struct {
		name string
		expr string
		want string
	}{
		{"int divide by zero", "1 / 0", "divide by zero"},
		{"int rem by zero", "1 % 0", "divide by zero"},
		{"kind mismatch add", `1 + "x"`, "mismatched kinds"},
		{"unary minus on bool", "-true", "numeric"},
		{"not on int", "!1", "bool"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ip, scope, x := subsetExprOf(t, c.expr)
			_, err := ip.evalExpr(x, scope)
			if err == nil {
				t.Fatalf("eval %q: want error, got nil", c.expr)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("eval %q: error %q does not contain %q", c.expr, err.Error(), c.want)
			}
		})
	}
}
