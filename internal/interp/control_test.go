package interp

// These tests prove US-008 "Eval control flow": the interpreter evaluates
// if/else, three-clause and condition-only for loops, switch (with cases and a
// default), nested block scoping, and break/continue. Programs are parsed and
// sema-resolved through the shared front-end (newInterp, defined in call_test.go)
// and exercised by calling a top-level function via evalExpr against the root
// scope — the standard direct-evalExpr testing seam from US-005..US-007. stdlib
// testing only (no testify), per the project's zero-dependency constraint.

import (
	"errors"
	"strings"
	"testing"

	"goal/internal/ast"
	"goal/internal/token"
)

// callInt parses src, calls the named no-arg function, and asserts it returns a
// single int equal to want.
func callInt(t *testing.T, src, name string, want int64) {
	t.Helper()
	ip := newInterp(t, src)
	got, err := ip.evalExpr(call(name), ip.root)
	if err != nil {
		t.Fatalf("%s(): unexpected error: %v", name, err)
	}
	if got.Kind != KindInt || got.Int != want {
		t.Fatalf("%s() = %s, want %d", name, got.String(), want)
	}
}

// TestForThreeClauseSummation proves a three-clause for loop with an `i++` post
// clause accumulates correctly: sum of 0..9 is 45.
func TestForThreeClauseSummation(t *testing.T) {
	src := `package main

func run() int {
	sum := 0
	for i := 0; i < 10; i++ {
		sum += i
	}
	return sum
}

func main() {}
`
	callInt(t, src, "run", 45)
}

// TestForConditionOnly proves a condition-only loop (`for cond`) runs until the
// condition goes false.
func TestForConditionOnly(t *testing.T) {
	src := `package main

func run() int {
	n := 0
	count := 0
	for n < 100 {
		n += 7
		count += 1
	}
	return count
}

func main() {}
`
	// 7*15 = 105 >= 100 after 15 iterations.
	callInt(t, src, "run", 15)
}

// TestForInfiniteWithBreak proves an infinite loop (`for {}`) terminated by break
// runs and stops at the right point.
func TestForInfiniteWithBreak(t *testing.T) {
	src := `package main

func run() int {
	i := 0
	for {
		if i >= 5 {
			break
		}
		i += 1
	}
	return i
}

func main() {}
`
	callInt(t, src, "run", 5)
}

// TestContinueSkipsRemainder proves continue skips the rest of the body and
// advances the post clause: summing only even i in 0..9 yields 20.
func TestContinueSkipsRemainder(t *testing.T) {
	src := `package main

func run() int {
	sum := 0
	for i := 0; i < 10; i++ {
		if i % 2 == 1 {
			continue
		}
		sum += i
	}
	return sum
}

func main() {}
`
	callInt(t, src, "run", 20)
}

// TestTaggedSwitchDispatch proves a tagged switch selects the case whose
// expression equals the tag.
func TestTaggedSwitchDispatch(t *testing.T) {
	src := `package main

func classify(n int) int {
	switch n {
	case 1:
		return 10
	case 2:
		return 20
	case 3:
		return 30
	default:
		return -1
	}
}

func main() {}
`
	ip := newInterp(t, src)
	cases := []struct {
		arg  string
		want int64
	}{
		{"1", 10}, {"2", 20}, {"3", 30}, {"9", -1},
	}
	for _, c := range cases {
		got, err := ip.evalExpr(call("classify", intLit(c.arg)), ip.root)
		if err != nil {
			t.Fatalf("classify(%s): %v", c.arg, err)
		}
		if got.Kind != KindInt || got.Int != c.want {
			t.Fatalf("classify(%s) = %s, want %d", c.arg, got.String(), c.want)
		}
	}
}

// TestTaglessSwitchFirstTrue proves a tagless switch selects the first case whose
// expression is true.
func TestTaglessSwitchFirstTrue(t *testing.T) {
	src := `package main

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n == 0:
		return 0
	default:
		return 1
	}
}

func main() {}
`
	ip := newInterp(t, src)
	cases := []struct {
		arg  string
		want int64
	}{
		{"-5", -1}, {"0", 0}, {"42", 1},
	}
	for _, c := range cases {
		got, err := ip.evalExpr(call("sign", intLit(c.arg)), ip.root)
		if err != nil {
			t.Fatalf("sign(%s): %v", c.arg, err)
		}
		if got.Kind != KindInt || got.Int != c.want {
			t.Fatalf("sign(%s) = %s, want %d", c.arg, got.String(), c.want)
		}
	}
}

// TestBreakInSwitchExitsOnlySwitch proves a break inside a switch ends the switch
// but not the enclosing loop, so the loop runs to completion.
func TestBreakInSwitchExitsOnlySwitch(t *testing.T) {
	src := `package main

func run() int {
	count := 0
	for i := 0; i < 4; i++ {
		switch i {
		case 2:
			break
		default:
			count += 1
		}
		count += 10
	}
	return count
}

func main() {}
`
	// i=0,1,3 hit default (+1) => 3; every iteration adds +10 after the switch
	// (the break only left the switch) => 40. Total 43.
	callInt(t, src, "run", 43)
}

// TestNestedBlockScoping proves a variable declared in a nested block does not
// leak into the enclosing scope.
func TestNestedBlockScoping(t *testing.T) {
	src := `package main

func main() {}
`
	ip := newInterp(t, src)
	scope := ip.root.NewChild()
	// Build `{ inner := 5 }` as a bare nested block and execute it.
	block := &ast.BlockStmt{List: []ast.Stmt{
		&ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: "inner"}},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{intLit("5")},
		},
	}}
	if err := ip.execStmt(block, scope); err != nil {
		t.Fatalf("execStmt(block): %v", err)
	}
	if _, err := scope.Lookup("inner"); err == nil {
		t.Fatalf("inner leaked out of the nested block into the enclosing scope")
	}
}

// TestIfElseChain is a regression guard that an if/else-if/else chain selects the
// correct branch (if/else was wired in an earlier story; control flow must not
// regress it).
func TestIfElseChain(t *testing.T) {
	src := `package main

func bucket(n int) int {
	if n < 0 {
		return -1
	} else if n == 0 {
		return 0
	} else {
		return 1
	}
}

func main() {}
`
	ip := newInterp(t, src)
	cases := []struct {
		arg  string
		want int64
	}{
		{"-3", -1}, {"0", 0}, {"7", 1},
	}
	for _, c := range cases {
		got, err := ip.evalExpr(call("bucket", intLit(c.arg)), ip.root)
		if err != nil {
			t.Fatalf("bucket(%s): %v", c.arg, err)
		}
		if got.Kind != KindInt || got.Int != c.want {
			t.Fatalf("bucket(%s) = %s, want %d", c.arg, got.String(), c.want)
		}
	}
}

// TestNonBoolForConditionErrors proves a non-bool for condition is a descriptive
// refusal rather than a silent run.
func TestNonBoolForConditionErrors(t *testing.T) {
	ip := newInterp(t, "package main\nfunc main() {}\n")
	scope := ip.root.NewChild()
	s := &ast.ForStmt{
		Cond: intLit("3"),
		Body: &ast.BlockStmt{},
	}
	err := ip.execStmt(s, scope)
	if err == nil || !strings.Contains(err.Error(), "for condition must be bool") {
		t.Fatalf("non-bool for condition: got err %v, want a 'must be bool' refusal", err)
	}
}

// TestBreakOutsideLoopErrors proves a break with no enclosing loop or switch
// surfaces as a descriptive error rather than a silent no-op.
func TestBreakOutsideLoopErrors(t *testing.T) {
	ip := newInterp(t, "package main\nfunc main() {}\n")
	scope := ip.root.NewChild()
	err := ip.execStmt(&ast.BranchStmt{Tok: token.BREAK}, scope)
	var brk breakSignal
	if !errors.As(err, &brk) {
		t.Fatalf("break outside loop: got %v, want a breakSignal to propagate", err)
	}
}
