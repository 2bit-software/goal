package interp

// These tests prove US-006: the interpreter evaluates declarations (var, short
// `:=`, const) and assignment (plain `=` and the arithmetic compound forms),
// reading and writing program state through the lexical Env. The interpreter
// runs the body of `func main` in a fresh child scope and the test reads the
// final bindings back out of that scope.

import (
	"errors"
	"testing"

	"goal/internal/parser"
	"goal/internal/sema"
)

// runMainScope parses + sema-resolves a program, runs the body of func main in
// a fresh child scope, and returns that scope so the test can read back the
// final variable bindings.
func runMainScope(t *testing.T, src string) *Env {
	t.Helper()
	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	info := sema.Resolve(file)
	ip := New(file, info)
	main := ip.findMain()
	if main == nil {
		t.Fatal("program declares no func main")
	}
	scope := ip.root.NewChild()
	if err := ip.execBlock(main.Body, scope); err != nil {
		t.Fatalf("execBlock: unexpected error: %v", err)
	}
	return scope
}

func mustLookup(t *testing.T, scope *Env, name string) Value {
	t.Helper()
	v, err := scope.Lookup(name)
	if err != nil {
		t.Fatalf("Lookup(%q): %v", name, err)
	}
	return v
}

// TestDeclareReassignCompound is the acceptance-criteria test: a program that
// declares variables with var (with and without an initializer), `:=`, and
// const, reassigns with `=`, and compound-assigns yields the expected final
// values.
func TestDeclareReassignCompound(t *testing.T) {
	const src = `package main
func main() {
	var a = 1
	var b int
	c := 2
	const d = 10
	a = a + d
	b += 5
	c -= 1
	c *= 4
	b %= 4
}
`
	scope := runMainScope(t, src)

	want := map[string]int64{
		"a": 11, // 1 + 10
		"b": 1,  // 0 += 5 -> 5; 5 %= 4 -> 1
		"c": 4,  // 2 -= 1 -> 1; 1 *= 4 -> 4
		"d": 10, // const, unchanged
	}
	for name, exp := range want {
		v := mustLookup(t, scope, name)
		if v.Kind != KindInt || v.Int != exp {
			t.Errorf("%s = %s, want int %d", name, v.String(), exp)
		}
	}
}

// TestVarWithoutInitializerZeroes proves a var with no initializer binds the
// declared type's safe zero value.
func TestVarWithoutInitializerZeroes(t *testing.T) {
	const src = `package main
func main() {
	var i int
	var f float64
	var s string
	var b bool
}
`
	scope := runMainScope(t, src)

	if v := mustLookup(t, scope, "i"); v.Kind != KindInt || v.Int != 0 {
		t.Errorf("i = %s, want int 0", v.String())
	}
	if v := mustLookup(t, scope, "f"); v.Kind != KindFloat || v.Float != 0 {
		t.Errorf("f = %s, want float 0", v.String())
	}
	if v := mustLookup(t, scope, "s"); v.Kind != KindString || v.Str != "" {
		t.Errorf("s = %s, want empty string", v.String())
	}
	if v := mustLookup(t, scope, "b"); v.Kind != KindBool || v.Bool {
		t.Errorf("b = %s, want false", v.String())
	}
}

// TestShortVarAndReadInExpression proves a `:=`-declared name reads back its
// value in a later expression.
func TestShortVarAndReadInExpression(t *testing.T) {
	const src = `package main
func main() {
	x := 3
	y := x * x
}
`
	scope := runMainScope(t, src)
	if v := mustLookup(t, scope, "y"); v.Kind != KindInt || v.Int != 9 {
		t.Errorf("y = %s, want int 9", v.String())
	}
}

// TestAssignUpdatesExistingBindingNotShadow proves plain `=` mutates the binding
// where the variable was declared (in a parent scope) rather than defining a new
// shadowing binding in the inner scope.
func TestAssignUpdatesExistingBindingNotShadow(t *testing.T) {
	parent := NewEnv()
	parent.Define("n", IntVal(1))
	child := parent.NewChild()

	if err := child.Assign("n", IntVal(42)); err != nil {
		t.Fatalf("Assign through child: %v", err)
	}
	// The change must be visible through the parent — not shadowed in child.
	if v := mustLookup(t, parent, "n"); v.Int != 42 {
		t.Errorf("parent n = %s, want 42 (assign updated the owning scope)", v.String())
	}
	if _, ok := child.vars["n"]; ok {
		t.Error("Assign created a shadowing binding in the child scope")
	}
}

// TestAssignUndeclaredErrors proves assigning to a name bound in no scope is a
// located, named error, not a silent define.
func TestAssignUndeclaredErrors(t *testing.T) {
	const src = `package main
func main() {
	missing = 5
}
`
	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	ip := New(file, sema.Resolve(file))
	main := ip.findMain()
	err = ip.execBlock(main.Body, ip.root.NewChild())
	if err == nil {
		t.Fatal("assign to undeclared name: want error, got nil")
	}
	var nf *NotFoundError
	if !errors.As(err, &nf) || nf.Name != "missing" {
		t.Fatalf("want NotFoundError naming \"missing\", got %v", err)
	}
}

// TestReadUndefinedErrors proves reading an undefined name is a located error.
func TestReadUndefinedErrors(t *testing.T) {
	const src = `package main
func main() {
	y := nope + 1
}
`
	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	ip := New(file, sema.Resolve(file))
	main := ip.findMain()
	err = ip.execBlock(main.Body, ip.root.NewChild())
	if err == nil {
		t.Fatal("read of undefined name: want error, got nil")
	}
	var nf *NotFoundError
	if !errors.As(err, &nf) || nf.Name != "nope" {
		t.Fatalf("want NotFoundError naming \"nope\", got %v", err)
	}
}
