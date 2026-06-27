package parser

import (
	"testing"

	"goal/internal/ast"
)

// findFunc returns the first FuncDecl named name, or fails.
func findFunc(t *testing.T, f *ast.File, name string) *ast.FuncDecl {
	t.Helper()
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name != nil && fd.Name.Name == name {
			return fd
		}
	}
	t.Fatalf("func %q not found", name)
	return nil
}

// variantName returns the variant tag of a VariantPattern, or fails.
func variantName(t *testing.T, e ast.Expr) string {
	t.Helper()
	vp, ok := e.(*ast.VariantPattern)
	if !ok {
		t.Fatalf("expected *ast.VariantPattern, got %T", e)
	}
	if vp.Variant == nil {
		t.Fatalf("variant pattern has no variant tag")
	}
	return vp.Variant.Name
}

// TestParseMatchStatementPosition parses a bare `match` inside a function body
// (statement position) and asserts it is an ExprStmt wrapping a MatchExpr with
// the expected variant/binding arms.
func TestParseMatchStatementPosition(t *testing.T) {
	src := readExample(t, "features/02-match/examples/status_match.goal")
	f, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	fd := findFunc(t, f, "handle")
	if fd.Body == nil || len(fd.Body.List) != 1 {
		t.Fatalf("handle body: want 1 statement, got %d", len(fd.Body.List))
	}
	es, ok := fd.Body.List[0].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("statement-position match: want *ast.ExprStmt, got %T", fd.Body.List[0])
	}
	m, ok := es.X.(*ast.MatchExpr)
	if !ok {
		t.Fatalf("want *ast.MatchExpr, got %T", es.X)
	}
	if len(m.Arms) != 3 {
		t.Fatalf("arms: want 3, got %d", len(m.Arms))
	}

	// Subject is the bare identifier `s`.
	if id, ok := m.Subject.(*ast.Ident); !ok || id.Name != "s" {
		t.Fatalf("subject: want ident s, got %#v", m.Subject)
	}

	// Arm 0: Status.Pending (data-less, no binding).
	if got := variantName(t, m.Arms[0].Pattern); got != "Pending" {
		t.Fatalf("arm 0 variant: want Pending, got %s", got)
	}
	if vp := m.Arms[0].Pattern.(*ast.VariantPattern); vp.Binding != nil {
		t.Fatalf("arm 0: want no binding, got %s", vp.Binding.Name)
	}
	if enum := m.Arms[0].Pattern.(*ast.VariantPattern).Enum; enum == nil {
		t.Fatalf("arm 0: want enum reference Status")
	}

	// Arm 1: Status.Active(a) — binding `a`.
	if got := variantName(t, m.Arms[1].Pattern); got != "Active" {
		t.Fatalf("arm 1 variant: want Active, got %s", got)
	}
	if vp := m.Arms[1].Pattern.(*ast.VariantPattern); vp.Binding == nil || vp.Binding.Name != "a" {
		t.Fatalf("arm 1: want binding a, got %#v", vp.Binding)
	}

	// Arm 2: Status.Cancelled(c) — binding `c`.
	if got := variantName(t, m.Arms[2].Pattern); got != "Cancelled" {
		t.Fatalf("arm 2 variant: want Cancelled, got %s", got)
	}
	if vp := m.Arms[2].Pattern.(*ast.VariantPattern); vp.Binding == nil || vp.Binding.Name != "c" {
		t.Fatalf("arm 2: want binding c, got %#v", vp.Binding)
	}
}

// firstMatchExpr walks the file and returns the first MatchExpr found, or fails.
func firstMatchExpr(t *testing.T, f *ast.File) *ast.MatchExpr {
	t.Helper()
	var found *ast.MatchExpr
	ast.Walk(visitorFunc(func(n ast.Node) bool {
		if found != nil {
			return false
		}
		if m, ok := n.(*ast.MatchExpr); ok {
			found = m
			return false
		}
		return true
	}), f)
	if found == nil {
		t.Fatalf("no *ast.MatchExpr found in file")
	}
	return found
}

// visitorFunc adapts a func into an ast.Visitor; descending when it returns true.
type visitorFunc func(ast.Node) bool

func (vf visitorFunc) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}
	if vf(n) {
		return vf
	}
	return nil
}

// TestParseMatchValuePositionVar parses `var d string = match s { … }` and
// asserts the value-position match parses with the expected arms.
func TestParseMatchValuePositionVar(t *testing.T) {
	src := readExample(t, "features/02-match/examples/status_var.goal")
	f, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	m := firstMatchExpr(t, f)
	if len(m.Arms) != 3 {
		t.Fatalf("arms: want 3, got %d", len(m.Arms))
	}
	if got := variantName(t, m.Arms[0].Pattern); got != "Pending" {
		t.Fatalf("arm 0 variant: want Pending, got %s", got)
	}
}

// TestParseMatchValuePositionReturn parses `return match s { … }` and asserts
// the value-position match parses with the expected arms.
func TestParseMatchValuePositionReturn(t *testing.T) {
	src := readExample(t, "features/02-match/examples/status_return.goal")
	f, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	fd := findFunc(t, f, "label")
	ret, ok := fd.Body.List[len(fd.Body.List)-1].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf("want trailing *ast.ReturnStmt, got %T", fd.Body.List[len(fd.Body.List)-1])
	}
	if len(ret.Results) != 1 {
		t.Fatalf("return results: want 1, got %d", len(ret.Results))
	}
	m, ok := ret.Results[0].(*ast.MatchExpr)
	if !ok {
		t.Fatalf("value-position match: want *ast.MatchExpr, got %T", ret.Results[0])
	}
	if len(m.Arms) != 3 {
		t.Fatalf("arms: want 3, got %d", len(m.Arms))
	}
}

// TestParseMatchRestPattern parses a match with a `_` catch-all arm and asserts
// the rest pattern is a distinct RestPattern node.
func TestParseMatchRestPattern(t *testing.T) {
	src := readExample(t, "features/02-match/examples/status_rest.goal")
	f, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	m := firstMatchExpr(t, f)
	if len(m.Arms) != 2 {
		t.Fatalf("arms: want 2, got %d", len(m.Arms))
	}
	last := m.Arms[len(m.Arms)-1].Pattern
	if _, ok := last.(*ast.RestPattern); !ok {
		t.Fatalf("last arm pattern: want *ast.RestPattern, got %T", last)
	}
	// The first arm is still a variant pattern with a binding.
	if got := variantName(t, m.Arms[0].Pattern); got != "Active" {
		t.Fatalf("arm 0 variant: want Active, got %s", got)
	}
}

// TestParseMatchStatementArmBody pins that a brace-less statement match-arm body
// (`Option.Some(u) => return true`) parses: the arm body is the parsed
// statement, not an expression. This is the grammar gap the parse-100%-of-corpus
// gate exposed in features/04-option/examples/option_exists.goal.
func TestParseMatchStatementArmBody(t *testing.T) {
	src := readExample(t, "features/04-option/examples/option_exists.goal")
	f, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	m := firstMatchExpr(t, f)
	if len(m.Arms) != 2 {
		t.Fatalf("arms: want 2, got %d", len(m.Arms))
	}
	if _, ok := m.Arms[0].Body.(*ast.ReturnStmt); !ok {
		t.Fatalf("arm 0 body: want *ast.ReturnStmt, got %T", m.Arms[0].Body)
	}
}
