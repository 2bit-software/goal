package parser

import (
	"testing"

	"goal/internal/ast"
)

// stmtRHS returns the single RHS expression of the n-th statement of fn's body,
// which must be an assignment/short-var with one RHS value.
func assignRHS(t *testing.T, fn *ast.FuncDecl, n int) ast.Expr {
	t.Helper()
	if fn.Body == nil || len(fn.Body.List) <= n {
		t.Fatalf("func %s: no statement %d", fn.Name.Name, n)
	}
	as, ok := fn.Body.List[n].(*ast.AssignStmt)
	if !ok {
		t.Fatalf("func %s stmt %d: want *ast.AssignStmt, got %T", fn.Name.Name, n, fn.Body.List[n])
	}
	if len(as.Rhs) != 1 {
		t.Fatalf("func %s stmt %d: want 1 RHS, got %d", fn.Name.Name, n, len(as.Rhs))
	}
	return as.Rhs[0]
}

// TestParseVariantLitLabeledArgs parses Status.Active(since: now()) and asserts
// it is an *ast.VariantLit (not a CallExpr) with the expected enum reference,
// variant tag, and labeled argument.
func TestParseVariantLitLabeledArgs(t *testing.T) {
	src := readExample(t, "features/01-enums/examples/status.goal")
	f, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}
	fn := findFunc(t, f, "examples")

	// a := Status.Active(since: now())  — the second statement.
	rhs := assignRHS(t, fn, 1)
	vl, ok := rhs.(*ast.VariantLit)
	if !ok {
		t.Fatalf("Status.Active(since: now()): want *ast.VariantLit, got %T", rhs)
	}
	if enum, ok := vl.Enum.(*ast.Ident); !ok || enum.Name != "Status" {
		t.Errorf("VariantLit.Enum: want Ident Status, got %T %v", vl.Enum, vl.Enum)
	}
	if vl.Variant == nil || vl.Variant.Name != "Active" {
		t.Errorf("VariantLit.Variant: want Active, got %v", vl.Variant)
	}
	if len(vl.Args) != 1 {
		t.Fatalf("VariantLit.Args: want 1, got %d", len(vl.Args))
	}
	la, ok := vl.Args[0].(*ast.LabeledArg)
	if !ok {
		t.Fatalf("VariantLit.Args[0]: want *ast.LabeledArg, got %T", vl.Args[0])
	}
	if la.Label == nil || la.Label.Name != "since" {
		t.Errorf("LabeledArg.Label: want since, got %v", la.Label)
	}
	if _, ok := la.Value.(*ast.CallExpr); !ok {
		t.Errorf("LabeledArg.Value: want *ast.CallExpr now(), got %T", la.Value)
	}

	// c := Status.Cancelled(reason: "timeout", at: now())  — two labeled args.
	rhs = assignRHS(t, fn, 2)
	vl, ok = rhs.(*ast.VariantLit)
	if !ok {
		t.Fatalf("Status.Cancelled(...): want *ast.VariantLit, got %T", rhs)
	}
	if len(vl.Args) != 2 {
		t.Fatalf("Cancelled VariantLit.Args: want 2, got %d", len(vl.Args))
	}
	for i, want := range []string{"reason", "at"} {
		la, ok := vl.Args[i].(*ast.LabeledArg)
		if !ok {
			t.Fatalf("Cancelled arg %d: want *ast.LabeledArg, got %T", i, vl.Args[i])
		}
		if la.Label == nil || la.Label.Name != want {
			t.Errorf("Cancelled arg %d label: want %s, got %v", i, want, la.Label)
		}
	}
}

// TestParsePositionalCallStaysCall ensures an all-positional call is NOT turned
// into a VariantLit — only labeled-argument calls are construction.
func TestParsePositionalCallStaysCall(t *testing.T) {
	src := "package p\nfunc f() { _ = g(1, 2) }"
	f, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}
	fn := findFunc(t, f, "f")
	as := fn.Body.List[0].(*ast.AssignStmt)
	if _, ok := as.Rhs[0].(*ast.CallExpr); !ok {
		t.Fatalf("g(1, 2): want *ast.CallExpr, got %T", as.Rhs[0])
	}
}

// TestParseSpreadDefaults parses a composite literal containing `...defaults`
// and asserts the trailing element is an *ast.SpreadElement over the `defaults`
// identifier.
func TestParseSpreadDefaults(t *testing.T) {
	src := readExample(t, "features/08-no-zero-value/examples/defaults_primitives.goal")
	f, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}
	fn := findFunc(t, f, "newMember")
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		t.Fatalf("newMember: want a return with one result, got %T", fn.Body.List[0])
	}
	cl, ok := ret.Results[0].(*ast.CompositeLit)
	if !ok {
		t.Fatalf("return value: want *ast.CompositeLit, got %T", ret.Results[0])
	}
	if len(cl.Elts) != 3 {
		t.Fatalf("composite elements: want 3 (name, role, ...defaults), got %d", len(cl.Elts))
	}
	spread, ok := cl.Elts[2].(*ast.SpreadElement)
	if !ok {
		t.Fatalf("last element: want *ast.SpreadElement, got %T", cl.Elts[2])
	}
	if id, ok := spread.X.(*ast.Ident); !ok || id.Name != "defaults" {
		t.Errorf("SpreadElement.X: want Ident defaults, got %T %v", spread.X, spread.X)
	}
}

// TestParseSpreadDeriveCall parses a `...derive(e)` spread and asserts the
// spread's expression is the `derive(e)` call.
func TestParseSpreadDeriveCall(t *testing.T) {
	src := "package p\nfunc f(e T) S { return S{ID: e.ID, ...derive(e)} }"
	f, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}
	fn := findFunc(t, f, "f")
	ret := fn.Body.List[0].(*ast.ReturnStmt)
	cl := ret.Results[0].(*ast.CompositeLit)
	if len(cl.Elts) != 2 {
		t.Fatalf("composite elements: want 2, got %d", len(cl.Elts))
	}
	spread, ok := cl.Elts[1].(*ast.SpreadElement)
	if !ok {
		t.Fatalf("last element: want *ast.SpreadElement, got %T", cl.Elts[1])
	}
	call, ok := spread.X.(*ast.CallExpr)
	if !ok {
		t.Fatalf("SpreadElement.X: want *ast.CallExpr derive(e), got %T", spread.X)
	}
	if id, ok := call.Fun.(*ast.Ident); !ok || id.Name != "derive" {
		t.Errorf("spread call fun: want Ident derive, got %T %v", call.Fun, call.Fun)
	}
}
