package interp

// These tests prove expression evaluation (US-005): primitive literals and the
// arithmetic, comparison, logical (short-circuiting), and unary operators each
// evaluate to the Value Go would compute. Programs are evaluated by reaching the
// single expression statement in `func main` and running it through evalExpr —
// the same seam execBlock drives.

import (
	"strings"
	"testing"

	"goal/internal/ast"
	"goal/internal/parser"
	"goal/internal/sema"
)

// exprOf parses `package main\nfunc main() { <expr> }` and returns the parsed
// expression and a ready interpreter + scope for evaluating it.
func exprOf(t *testing.T, expr string) (*Interp, *Env, ast.Expr) {
	t.Helper()
	src := "package main\nfunc main() {\n" + expr + "\n}\n"
	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile(%q): %v", expr, err)
	}
	info := sema.Resolve(file)
	ip := New(file, info)
	main := ip.findMain()
	if main == nil || main.Body == nil || len(main.Body.List) == 0 {
		t.Fatalf("expr %q: no statement in main", expr)
	}
	es, ok := main.Body.List[0].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("expr %q: first statement is %T, want *ast.ExprStmt", expr, main.Body.List[0])
	}
	return ip, ip.root.NewChild(), es.X
}

// evalProgram evaluates a single expression program and returns its Value,
// failing the test on any evaluation error.
func evalProgram(t *testing.T, expr string) Value {
	t.Helper()
	ip, scope, x := exprOf(t, expr)
	v, err := ip.evalExpr(x, scope)
	if err != nil {
		t.Fatalf("evalExpr(%q): unexpected error: %v", expr, err)
	}
	return v
}

func TestEvalExpressions(t *testing.T) {
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
		{"int sub mul precedence", "2 + 3 * 4", IntVal(14)},
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
			got := evalProgram(t, c.expr)
			if !got.Equal(c.want) {
				t.Fatalf("eval %q = %s, want %s", c.expr, got.String(), c.want.String())
			}
		})
	}
}

// TestShortCircuit proves the right operand is not evaluated when the left
// operand already decides the result: a divide-by-zero on the right side would
// error if evaluated, but the expressions evaluate cleanly.
func TestShortCircuit(t *testing.T) {
	if got := evalProgram(t, "false && (1 / 0 == 0)"); !got.Equal(BoolVal(false)) {
		t.Fatalf("false && ... = %s, want false (right side must not be evaluated)", got.String())
	}
	if got := evalProgram(t, "true || (1 / 0 == 0)"); !got.Equal(BoolVal(true)) {
		t.Fatalf("true || ... = %s, want true (right side must not be evaluated)", got.String())
	}
}

// TestShortCircuitDoesEvaluateRightWhenNeeded confirms the right operand IS
// evaluated when the left does not decide the result (so the error surfaces).
func TestShortCircuitDoesEvaluateRightWhenNeeded(t *testing.T) {
	ip, scope, x := exprOf(t, "true && (1 / 0 == 0)")
	if _, err := ip.evalExpr(x, scope); err == nil {
		t.Fatal("true && (1/0==0): want divide-by-zero error, got nil")
	}
}

func TestEvalErrors(t *testing.T) {
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
			ip, scope, x := exprOf(t, c.expr)
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

// TestExecBlockEvaluatesExprStmt proves the statement-dispatch seam runs an
// expression statement through evalExpr (a failing expression surfaces; a
// well-formed one is a clean no-op-for-effect).
func TestExecBlockEvaluatesExprStmt(t *testing.T) {
	src := "package main\nfunc main() {\n1 / 0\n}\n"
	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	ip := New(file, sema.Resolve(file))
	if err := ip.Run(); err == nil {
		t.Fatal("Run with `1 / 0` expression statement: want error, got nil")
	}
}
