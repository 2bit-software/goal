package backend_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"goal/internal/backend"
)

// SEAM-CAP-3d: a 2-level sealed hierarchy. `sealed interface Expr` embeds
// `sealed interface Node`; concrete types declare `implements Expr` (Num, Neg) or
// `implements Node` directly (Comment). The embedding cascade must emit BOTH
// markers (isExpr() and isNode()) for an Expr implementor so the emitted Go
// compiles, and register it under both interfaces so `match` over either level is
// exhaustive. This is the AST topology SEAM-004 needs (Expr/Stmt/Decl/Spec embed
// Node).
const nestedSealedSrc = `package shape

sealed interface Node {}
sealed interface Expr { Node }

type Num struct implements Expr {
	Val int
}

type Neg struct implements Expr {
	Inner Expr
}

type Comment struct implements Node {
	Text string
}

func evalExpr(e Expr) int {
	return match e {
		*Num(n) => n.Val
		*Neg(g) => -evalExpr(g.Inner)
	}
}

func evalNode(n Node) int {
	return match n {
		*Num(x) => x.Val
		*Neg(g) => -evalExpr(g.Inner)
		*Comment(c) => 0
	}
}
`

// TestNestedSealedEmitsCascadedMarkers proves an Expr implementor emits BOTH its
// own marker (isExpr) and the embedded interface's marker (isNode), while a
// direct-Node implementor emits only isNode.
func TestNestedSealedEmitsCascadedMarkers(t *testing.T) {
	out, err := backend.Transpile(nestedSealedSrc)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	for _, want := range []string{
		"func (Num) isExpr() {}",
		"func (Num) isNode() {}", // cascaded through the embedded Node
		"func (Neg) isExpr() {}",
		"func (Neg) isNode() {}", // cascaded
		"func (Comment) isNode() {}",
		"case *Num:",
		"case *Comment:",
		".(type) {",
	} {
		if !strings.Contains(out.Go, want) {
			t.Errorf("transpiled Go missing %q:\n%s", want, out.Go)
		}
	}
	// A direct-Node type must NOT acquire an isExpr marker.
	if strings.Contains(out.Go, "func (Comment) isExpr() {}") {
		t.Errorf("direct-Node *Comment must not implement Expr:\n%s", out.Go)
	}
}

// TestNestedSealedBuildsAndBehaves builds the transpiled package into a throwaway
// `module goal` and runs it against a reference type-switch over BOTH levels. A
// clean build proves the cascade markers satisfy the embedding; the behavior test
// proves match over Expr and over Node behave identically to the type-switch.
func TestNestedSealedBuildsAndBehaves(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	out, err := backend.Transpile(nestedSealedSrc)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module goal\n\ngo 1.26\n")
	writeFile(t, filepath.Join(dir, "shape", "shape.go"), out.Go)
	writeFile(t, filepath.Join(dir, "shape", "behavior_test.go"), nestedSealedBehaviorTest)

	cmd := exec.Command("go", "test", "./shape/")
	cmd.Dir = dir
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("transpiled nested sealed hierarchy did not build/behave like the reference type-switch:\n%s", b)
	}
}

const nestedSealedBehaviorTest = `package shape

import "testing"

func refExpr(e Expr) int {
	switch x := e.(type) {
	case *Num:
		return x.Val
	case *Neg:
		return -refExpr(x.Inner)
	default:
		panic("unreachable")
	}
}

func refNode(n Node) int {
	switch x := n.(type) {
	case *Num:
		return x.Val
	case *Neg:
		return -refExpr(x.Inner)
	case *Comment:
		return 0
	default:
		panic("unreachable")
	}
}

func TestNestedEvalMatchesReference(t *testing.T) {
	// An Expr value is also a Node (embedding) — both eval funcs must agree.
	exprs := []Expr{
		&Num{Val: 5},
		&Neg{Inner: &Num{Val: 5}},
		&Neg{Inner: &Neg{Inner: &Num{Val: 2}}},
	}
	for _, e := range exprs {
		if got, want := evalExpr(e), refExpr(e); got != want {
			t.Errorf("evalExpr(%T) = %d, ref = %d", e, got, want)
		}
		if got, want := evalNode(e), refNode(e); got != want {
			t.Errorf("evalNode(%T) = %d, ref = %d", e, got, want)
		}
	}
	// A direct-Node value is matched only by evalNode.
	nodes := []Node{
		&Comment{Text: "x"},
		&Num{Val: 9},
	}
	for _, n := range nodes {
		if got, want := evalNode(n), refNode(n); got != want {
			t.Errorf("evalNode(%T) = %d, ref = %d", n, got, want)
		}
	}
}
`
