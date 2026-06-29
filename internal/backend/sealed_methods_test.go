package backend_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"goal/internal/backend"
)

// sealedWithMethodsSrc is a sealed interface that DECLARES methods (Pos/End),
// an implementor that provides them, and a function that calls one through the
// interface value. Before SEAM-CAP-3a the emitter dropped the declared methods,
// emitting only `type Node interface{ isNode() }` — sealing a method-bearing
// interface like ast.Node was therefore impossible.
const sealedWithMethodsSrc = `package node

type Position struct {
	Line int
	Col  int
}

type Leaf struct implements Node {
	P Position
}

func (l Leaf) Pos() Position { return l.P }
func (l Leaf) End() Position { return l.P }

sealed interface Node {
	Pos() Position
	End() Position
}

func startLine(n Node) int {
	return n.Pos().Line
}
`

// TestSealedInterfacePreservesMethodSignatures is SEAM-CAP-3a AC1/AC2: a sealed
// interface that declares methods keeps those signatures in the emitted Go
// interface, ALONGSIDE the synthesized marker method.
func TestSealedInterfacePreservesMethodSignatures(t *testing.T) {
	out, err := backend.Transpile(sealedWithMethodsSrc)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	for _, want := range []string{
		"Pos() Position", // declared method preserved
		"End() Position", // declared method preserved
		"isNode()",       // marker method still emitted
	} {
		if !strings.Contains(out.Go, want) {
			t.Errorf("emitted Go missing %q:\n%s", want, out.Go)
		}
	}
	// Guard against the old behavior: the compact marker-only form must NOT be
	// what we emit when methods are declared.
	if strings.Contains(out.Go, "interface{ isNode() }") {
		t.Errorf("sealed interface with methods emitted the compact marker-only form:\n%s", out.Go)
	}
}

// TestSealedInterfaceEmptyBodyStaysCompact is SEAM-CAP-3a AC (FR-3): an
// empty-body sealed interface keeps the compact marker-only form byte-identical
// to the prior lowering, so self-host fixpoint is unaffected.
func TestSealedInterfaceEmptyBodyStaysCompact(t *testing.T) {
	out, err := backend.Transpile("package shape\n\nsealed interface Shape {}\n")
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	if !strings.Contains(out.Go, "type Shape interface{ isShape() }") {
		t.Errorf("empty sealed interface lost its compact form:\n%s", out.Go)
	}
}

// TestSealedInterfaceMethodsCallableThroughInterface is SEAM-CAP-3a AC3: an
// implementor satisfying the declared methods compiles and those methods are
// callable through a value of the sealed-interface type. The transpiled package
// is built and tested in a throwaway `module goal`; if the emitted interface had
// dropped Pos/End, the call through the interface value would fail to compile.
func TestSealedInterfaceMethodsCallableThroughInterface(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	out, err := backend.Transpile(sealedWithMethodsSrc)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module goal\n\ngo 1.26\n")
	writeFile(t, filepath.Join(dir, "node", "node.go"), out.Go)
	writeFile(t, filepath.Join(dir, "node", "behavior_test.go"), sealedMethodsBehaviorTest)

	cmd := exec.Command("go", "test", "./node/")
	cmd.Dir = dir
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sealed-interface methods not preserved/callable through the interface:\n%s", b)
	}
}

// sealedMethodsBehaviorTest calls the declared methods through a value of the
// sealed-interface type — the proof that the signatures survived lowering.
const sealedMethodsBehaviorTest = `package node

import "testing"

func TestMethodsCallableThroughInterface(t *testing.T) {
	var n Node = Leaf{P: Position{Line: 7, Col: 3}}
	if got := n.Pos(); got.Line != 7 {
		t.Errorf("n.Pos().Line = %d, want 7", got.Line)
	}
	if got := n.End(); got.Col != 3 {
		t.Errorf("n.End().Col = %d, want 3", got.Col)
	}
	if got := startLine(n); got != 7 {
		t.Errorf("startLine(n) = %d, want 7", got)
	}
}
`
