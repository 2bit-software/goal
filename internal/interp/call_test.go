package interp

// These tests prove US-007 "Eval functions and calls": the interpreter evaluates
// function declarations as callable values, binds arguments to parameters in a
// fresh per-call scope, supports multiple return values, and supports recursion.
// Top-level functions are registered into the root scope by New, so a call
// against the root scope (the standard direct-evalExpr testing seam established by
// US-005/US-006) resolves the function — including recursively from its own body.

import (
	"errors"
	"strings"
	"testing"

	"goal/internal/ast"
	"goal/internal/parser"
	"goal/internal/sema"
	"goal/internal/token"
)

// newInterp parses + sema-resolves a goal program and constructs the interpreter
// (which registers its top-level functions in the root scope).
func newInterp(t *testing.T, src string) *Interp {
	t.Helper()
	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return New(file, sema.Resolve(file))
}

// intLit builds an integer literal expression.
func intLit(v string) ast.Expr { return &ast.BasicLit{Kind: token.INT, Value: v} }

// call builds a call of the named function with the given argument expressions.
func call(name string, args ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{Fun: &ast.Ident{Name: name}, Args: args}
}

const recursionProgram = `package main

func factorial(n int) int {
	if n <= 1 {
		return 1
	}
	return n * factorial(n - 1)
}

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n - 1) + fib(n - 2)
}

func divmod(a int, b int) (int, int) {
	return a / b, a % b
}

func main() {}
`

func TestRecursiveFactorial(t *testing.T) {
	ip := newInterp(t, recursionProgram)

	cases := []struct {
		n    string
		want int64
	}{
		{"0", 1}, {"1", 1}, {"5", 120}, {"6", 720},
	}
	for _, c := range cases {
		got, err := ip.evalExpr(call("factorial", intLit(c.n)), ip.root)
		if err != nil {
			t.Fatalf("factorial(%s): unexpected error: %v", c.n, err)
		}
		if got.Kind != KindInt || got.Int != c.want {
			t.Fatalf("factorial(%s) = %s, want %d", c.n, got.String(), c.want)
		}
	}
}

func TestRecursiveFibonacci(t *testing.T) {
	ip := newInterp(t, recursionProgram)

	cases := []struct {
		n    string
		want int64
	}{
		{"0", 0}, {"1", 1}, {"7", 13}, {"10", 55},
	}
	for _, c := range cases {
		got, err := ip.evalExpr(call("fib", intLit(c.n)), ip.root)
		if err != nil {
			t.Fatalf("fib(%s): unexpected error: %v", c.n, err)
		}
		if got.Kind != KindInt || got.Int != c.want {
			t.Fatalf("fib(%s) = %s, want %d", c.n, got.String(), c.want)
		}
	}
}

// TestMultiReturnAssignment proves a multi-value call spreads positionally into a
// multi-target short-var declaration: `q, r := divmod(17, 5)`.
func TestMultiReturnAssignment(t *testing.T) {
	ip := newInterp(t, recursionProgram)

	s := &ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: "q"}, &ast.Ident{Name: "r"}},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{call("divmod", intLit("17"), intLit("5"))},
	}
	scope := ip.root.NewChild()
	if err := ip.execAssign(s, scope); err != nil {
		t.Fatalf("execAssign q, r := divmod(17, 5): %v", err)
	}
	q, err := scope.Lookup("q")
	if err != nil {
		t.Fatalf("lookup q: %v", err)
	}
	r, err := scope.Lookup("r")
	if err != nil {
		t.Fatalf("lookup r: %v", err)
	}
	if q.Kind != KindInt || q.Int != 3 {
		t.Fatalf("q = %s, want 3", q.String())
	}
	if r.Kind != KindInt || r.Int != 2 {
		t.Fatalf("r = %s, want 2", r.String())
	}
}

// TestMultiValueCallInSingleValueContext proves a multi-value call is refused
// when used where exactly one value is expected.
func TestMultiValueCallInSingleValueContext(t *testing.T) {
	ip := newInterp(t, recursionProgram)

	_, err := ip.evalExpr(call("divmod", intLit("17"), intLit("5")), ip.root)
	if err == nil {
		t.Fatal("multi-value call in single-value context: want error, got nil")
	}
	if !strings.Contains(err.Error(), "multi-value") {
		t.Fatalf("error %q does not describe the multi-value misuse", err.Error())
	}
}

// TestArgCountMismatch proves an argument-count mismatch is a descriptive,
// named refusal rather than a silent wrong result.
func TestArgCountMismatch(t *testing.T) {
	ip := newInterp(t, recursionProgram)

	_, err := ip.evalExpr(call("factorial"), ip.root) // no args, factorial wants 1
	if err == nil {
		t.Fatal("arg-count mismatch: want error, got nil")
	}
	if !strings.Contains(err.Error(), "factorial") ||
		!strings.Contains(err.Error(), "1") || !strings.Contains(err.Error(), "0") {
		t.Fatalf("error %q does not name the function and the expected/actual counts", err.Error())
	}
}

// TestCallNonFunction proves calling a non-function value is a descriptive
// refusal naming the offending kind.
func TestCallNonFunction(t *testing.T) {
	ip := newInterp(t, "package main\nfunc main() {}\n")

	scope := ip.root.NewChild()
	scope.Define("x", IntVal(7))
	_, err := ip.evalExpr(call("x"), scope)
	if err == nil {
		t.Fatal("call of non-function: want error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot call") || !strings.Contains(err.Error(), "int") {
		t.Fatalf("error %q does not describe the non-function call", err.Error())
	}
}

// TestUndefinedCall proves calling an undefined name surfaces the located
// *NotFoundError.
func TestUndefinedCall(t *testing.T) {
	ip := newInterp(t, "package main\nfunc main() {}\n")

	_, err := ip.evalExpr(call("nope", intLit("1")), ip.root)
	if err == nil {
		t.Fatal("undefined call: want error, got nil")
	}
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("undefined call: want *NotFoundError, got %T (%v)", err, err)
	}
	if nf.Name != "nope" {
		t.Fatalf("NotFoundError names %q, want \"nope\"", nf.Name)
	}
}

// TestFunctionDeclIsAValue proves a top-level function declaration is registered
// as a callable function value in the root scope.
func TestFunctionDeclIsAValue(t *testing.T) {
	ip := newInterp(t, recursionProgram)

	v, err := ip.root.Lookup("factorial")
	if err != nil {
		t.Fatalf("lookup factorial: %v", err)
	}
	if v.Kind != KindFunc || v.Func == nil || v.Func.Decl == nil {
		t.Fatalf("factorial bound as %s, want a callable function value", v.String())
	}
	if v.Func.Name != "factorial" {
		t.Fatalf("function value name = %q, want \"factorial\"", v.Func.Name)
	}
}
