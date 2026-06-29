package ast

import (
	"testing"

	"goal/internal/token"
)

// TestFuncDeclModifier covers Walk descent and Pos() behavior for a FuncDecl
// carrying a goal `from`/`derive`/plain modifier (FuncMod).
//
// This lives in its own internal-only test file (NOT in the self-host port-gate
// test list at internal/selfhost/port_test.go) because it references the
// Go-iota FuncMod constants (FuncFrom/FuncDerive/FuncPlain) directly. In the
// self-hosted tree (SEAM-002) FuncMod is a goal `enum`, whose transpiled Go has
// no such bare constants — so these assertions cannot be compiled against the
// enum form. The bootstrap reference compiler's own AST (this package) stays
// Go-iota, and keeping this coverage here lets the shared ast_test.go remain
// FuncMod/ChanDir-symbol-free so it compiles against both representations.
//
// collect/assertChildren are re-created locally: in ast_test.go they are
// function-local closures of TestWalkGoalDeclChildren, not package-level helpers.
func TestFuncDeclModifier(t *testing.T) {
	collect := func(root Node) map[Node]int {
		c := &collector{visits: make(map[Node]int)}
		Walk(c, root)
		return c.visits
	}
	assertChildren := func(t *testing.T, visits map[Node]int, parent Node, children ...Node) {
		t.Helper()
		if got := visits[parent]; got != 1 {
			t.Errorf("parent %T visited %d times, want 1", parent, got)
		}
		for _, ch := range children {
			if got := visits[ch]; got != 1 {
				t.Errorf("Walk did not descend from %T into child %T (visited %d times, want 1)", parent, ch, got)
			}
		}
	}

	// from func uuidToString(u UUID) string { ... }
	fromName := &Ident{Name: "uuidToString"}
	modPos := token.Pos{Offset: 0, Line: 1, Col: 1}
	fromFunc := &FuncDecl{Mod: FuncFrom, ModPos: modPos, Name: fromName, Type: &FuncType{}}
	{
		visits := collect(fromFunc)
		assertChildren(t, visits, fromFunc, fromName)
		if fromFunc.Mod != FuncFrom {
			t.Errorf("from-func Mod = %v, want FuncFrom", fromFunc.Mod)
		}
		if fromFunc.Pos() != modPos {
			t.Errorf("from-func Pos() = %v, want ModPos %v", fromFunc.Pos(), modPos)
		}
	}

	// derive func fromStorage(s StoredEvent) (EventExecution, error)  — bodyless
	deriveName := &Ident{Name: "fromStorage"}
	deriveFunc := &FuncDecl{Mod: FuncDerive, ModPos: modPos, Name: deriveName, Type: &FuncType{}}
	{
		visits := collect(deriveFunc)
		assertChildren(t, visits, deriveFunc, deriveName)
		if deriveFunc.Mod != FuncDerive {
			t.Errorf("derive-func Mod = %v, want FuncDerive", deriveFunc.Mod)
		}
		if deriveFunc.Pos() != modPos {
			t.Errorf("derive-func Pos() = %v, want ModPos %v", deriveFunc.Pos(), modPos)
		}
	}

	// A plain FuncDecl must NOT report ModPos as its start position.
	plainName := &Ident{Name: "plain"}
	plainFunc := &FuncDecl{Name: plainName, Type: &FuncType{Func: token.Pos{Offset: 5, Line: 1, Col: 6}}}
	if plainFunc.Mod != FuncPlain {
		t.Errorf("plain-func Mod = %v, want FuncPlain", plainFunc.Mod)
	}
	if plainFunc.Pos() != plainFunc.Type.Pos() {
		t.Errorf("plain-func Pos() = %v, want Type.Pos() %v", plainFunc.Pos(), plainFunc.Type.Pos())
	}
}
