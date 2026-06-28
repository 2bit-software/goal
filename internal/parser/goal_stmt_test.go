package parser

import (
	"strings"
	"testing"

	"goal/internal/ast"
	"goal/internal/token"
)

// parseExample parses a feature example .goal input and fails on any parse error.
func parseExample(t *testing.T, rel string) *ast.File {
	t.Helper()
	f, err := ParseFile(readExample(t, rel))
	if err != nil {
		t.Fatalf("ParseFile(%s) returned errors: %v", rel, err)
	}
	return f
}

// bodyAsserts returns the AssertStmt statements directly in a function body.
func bodyAsserts(fd *ast.FuncDecl) []*ast.AssertStmt {
	var out []*ast.AssertStmt
	if fd.Body == nil {
		return out
	}
	for _, s := range fd.Body.List {
		if a, ok := s.(*ast.AssertStmt); ok {
			out = append(out, a)
		}
	}
	return out
}

// --- FR-2: assert statements ----------------------------------------------

func TestParseAssertBare(t *testing.T) {
	f := parseExample(t, "features/10-assert/examples/bank.goal")
	asserts := bodyAsserts(findFunc(t, f, "withdraw"))
	if len(asserts) != 1 {
		t.Fatalf("withdraw: got %d asserts, want 1", len(asserts))
	}
	a := asserts[0]
	if a.Cond == nil {
		t.Errorf("bare assert: Cond is nil")
	}
	if a.Msg != nil || len(a.Args) != 0 {
		t.Errorf("bare assert: Msg=%v Args=%d, want nil/0", a.Msg, len(a.Args))
	}
	if _, ok := a.Cond.(*ast.BinaryExpr); !ok {
		t.Errorf("bare assert: Cond = %T, want *ast.BinaryExpr (amount > 0)", a.Cond)
	}
}

func TestParseAssertMessage(t *testing.T) {
	f := parseExample(t, "features/10-assert/examples/message.goal")
	asserts := bodyAsserts(findFunc(t, f, "setAge"))
	if len(asserts) != 1 {
		t.Fatalf("setAge: got %d asserts, want 1", len(asserts))
	}
	a := asserts[0]
	lit, ok := a.Msg.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		t.Fatalf("message assert: Msg = %T, want a string *ast.BasicLit", a.Msg)
	}
	if !strings.Contains(lit.Value, "non-negative") {
		t.Errorf("message assert: Msg.Value = %q, want it to contain the format text", lit.Value)
	}
	if len(a.Args) != 1 {
		t.Errorf("message assert: got %d args, want 1 (age)", len(a.Args))
	}
}

func TestParseAssertTopLevelCommaSplit(t *testing.T) {
	f := parseExample(t, "features/10-assert/examples/multiple.goal")
	asserts := bodyAsserts(findFunc(t, f, "check"))
	if len(asserts) != 3 {
		t.Fatalf("check: got %d asserts, want 3", len(asserts))
	}

	// 1st: `assert lo <= hi` — bare.
	if asserts[0].Msg != nil || len(asserts[0].Args) != 0 {
		t.Errorf("assert[0]: want bare, got Msg=%v Args=%d", asserts[0].Msg, len(asserts[0].Args))
	}

	// 2nd: `assert clamp(lo, hi, n) == n, "...", n, lo, hi` — the message split
	// must fire only on the TOP-LEVEL comma: the call's internal commas stay in
	// the condition, so Cond is a BinaryExpr whose LHS is the call.
	a := asserts[1]
	bin, ok := a.Cond.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("assert[1]: Cond = %T, want *ast.BinaryExpr", a.Cond)
	}
	call, ok := bin.X.(*ast.CallExpr)
	if !ok {
		t.Fatalf("assert[1]: Cond.X = %T, want *ast.CallExpr (clamp(lo, hi, n))", bin.X)
	}
	if len(call.Args) != 3 {
		t.Errorf("assert[1]: clamp call has %d args, want 3 (internal commas absorbed)", len(call.Args))
	}
	if a.Msg == nil {
		t.Errorf("assert[1]: Msg is nil, want the format string")
	}
	if len(a.Args) != 3 {
		t.Errorf("assert[1]: got %d format args, want 3 (n, lo, hi)", len(a.Args))
	}

	// 3rd: `assert n%2 == 0` — bare; the `%` is a remainder op, not a format string.
	if asserts[2].Msg != nil {
		t.Errorf("assert[2]: Msg = %v, want nil (n%%2 == 0 is bare)", asserts[2].Msg)
	}
}

// --- FR-1: from/derive function declarations -------------------------------

func TestParseFromDerive(t *testing.T) {
	// from_storage.goal: bodied `from func parseUUID`, bodyless `derive func fromStorage`.
	f := parseExample(t, "features/12-derive-convert/examples/from_storage.goal")
	parseUUID := findFunc(t, f, "parseUUID")
	if parseUUID.Mod != ast.FuncFrom {
		t.Errorf("parseUUID: Mod = %v, want FuncFrom", parseUUID.Mod)
	}
	if parseUUID.Body == nil {
		t.Errorf("parseUUID: Body is nil, want a bodied from func")
	}
	fromStorage := findFunc(t, f, "fromStorage")
	if fromStorage.Mod != ast.FuncDerive {
		t.Errorf("fromStorage: Mod = %v, want FuncDerive", fromStorage.Mod)
	}
	if fromStorage.Body != nil {
		t.Errorf("fromStorage: Body != nil, want a bodyless derive func")
	}

	// slice.goal: bodyless `derive func toIDs`, bodied `from func uuidToString`.
	f = parseExample(t, "features/12-derive-convert/examples/slice.goal")
	toIDs := findFunc(t, f, "toIDs")
	if toIDs.Mod != ast.FuncDerive {
		t.Errorf("toIDs: Mod = %v, want FuncDerive", toIDs.Mod)
	}
	if toIDs.Body != nil {
		t.Errorf("toIDs: Body != nil, want a bodyless derive func")
	}
	if u := findFunc(t, f, "uuidToString"); u.Mod != ast.FuncFrom || u.Body == nil {
		t.Errorf("uuidToString: Mod=%v Body=%v, want FuncFrom + bodied", u.Mod, u.Body)
	}

	// to_storage.goal: bodied `from func`s and a bodied `derive func toStorage`.
	f = parseExample(t, "features/12-derive-convert/examples/to_storage.goal")
	toStorage := findFunc(t, f, "toStorage")
	if toStorage.Mod != ast.FuncDerive {
		t.Errorf("toStorage: Mod = %v, want FuncDerive", toStorage.Mod)
	}
	if toStorage.Body == nil {
		t.Errorf("toStorage: Body is nil, want a bodied derive func")
	}
	if pn := findFunc(t, f, "ptrToNull"); pn.Mod != ast.FuncFrom {
		t.Errorf("ptrToNull: Mod = %v, want FuncFrom", pn.Mod)
	}
}

// --- FR-3: /// doctest comments --------------------------------------------

func TestParseDoctests(t *testing.T) {
	// add.goal: one func with one doctest.
	f := parseExample(t, "features/11-doctests/examples/add.goal")
	add := findFunc(t, f, "add")
	if add.Doc == nil {
		t.Fatalf("add: Doc is nil, want an attached doc comment")
	}
	if len(add.Doc.Doctests) != 1 {
		t.Fatalf("add: got %d doctests, want 1", len(add.Doc.Doctests))
	}
	dt := add.Doc.Doctests[0]
	if !strings.Contains(dt.Input, "add(2, 3)") {
		t.Errorf("add doctest: Input = %q, want it to contain add(2, 3)", dt.Input)
	}
	if len(dt.Expected) != 1 || strings.TrimSpace(dt.Expected[0]) != "5" {
		t.Errorf("add doctest: Expected = %v, want [5]", dt.Expected)
	}

	// multi.goal: one func with two doctests.
	f = parseExample(t, "features/11-doctests/examples/multi.goal")
	repeat := findFunc(t, f, "repeat")
	if repeat.Doc == nil || len(repeat.Doc.Doctests) != 2 {
		t.Errorf("repeat: doctests = %v, want 2", repeat.Doc)
	}

	// mixed.goal: half has a doc with zero doctests; double has one.
	f = parseExample(t, "features/11-doctests/examples/mixed.goal")
	half := findFunc(t, f, "half")
	if half.Doc == nil {
		t.Fatalf("half: Doc is nil, want a prose-only doc comment")
	}
	if len(half.Doc.Doctests) != 0 {
		t.Errorf("half: got %d doctests, want 0 (prose only)", len(half.Doc.Doctests))
	}
	double := findFunc(t, f, "double")
	if double.Doc == nil || len(double.Doc.Doctests) != 1 {
		t.Errorf("double: doctests = %v, want 1", double.Doc)
	}
}

// --- Walk coverage of the new nodes ----------------------------------------

type counter struct{ seen map[ast.Node]int }

func (c *counter) Visit(n ast.Node) ast.Visitor {
	if n != nil {
		c.seen[n]++
	}
	return c
}

func TestWalkNewNodes(t *testing.T) {
	f := parseExample(t, "features/10-assert/examples/message.goal")
	c := &counter{seen: map[ast.Node]int{}}
	ast.Walk(c, f)

	a := bodyAsserts(findFunc(t, f, "setAge"))[0]
	if c.seen[a] != 1 {
		t.Errorf("AssertStmt visited %d times, want 1", c.seen[a])
	}
	if c.seen[a.Cond] != 1 {
		t.Errorf("AssertStmt.Cond visited %d times, want 1", c.seen[a.Cond])
	}
	if c.seen[a.Msg] != 1 {
		t.Errorf("AssertStmt.Msg visited %d times, want 1", c.seen[a.Msg])
	}
	for i, arg := range a.Args {
		if c.seen[arg] != 1 {
			t.Errorf("AssertStmt.Args[%d] visited %d times, want 1", i, c.seen[arg])
		}
	}

	// A FuncDecl.Doc is walked when present.
	fd := parseExample(t, "features/11-doctests/examples/add.goal")
	c2 := &counter{seen: map[ast.Node]int{}}
	ast.Walk(c2, fd)
	add := findFunc(t, fd, "add")
	if c2.seen[add.Doc] != 1 {
		t.Errorf("FuncDecl.Doc visited %d times, want 1", c2.seen[add.Doc])
	}
}
