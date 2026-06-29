package backend_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"goal/internal/backend"
)

// sealedMatchSrc is a same-package sealed-interface `match` (SEAM-CAP-3b): Node is a
// sealed interface with two concrete implementors (*Lit, *Neg), and eval matches over
// the concrete implementor types — the type-pattern form that replaces a
// `switch x := n.(type)`.
const sealedMatchSrc = `package shape

sealed interface Node {}

type Lit struct implements Node {
	Val int
}

type Neg struct implements Node {
	Inner Node
}

func eval(n Node) int {
	return match n {
		*Lit(l) => l.Val
		*Neg(g) => -eval(g.Inner)
	}
}
`

// TestSealedMatchLowersToTypeSwitch is the SEAM-CAP-3b lowering regression: a `match`
// over a same-package sealed interface lowers to a Go type-switch with concrete
// `case *T:` labels (the sealedMatch path), distinct from the §8.1 `Enum_Variant`
// enum path.
func TestSealedMatchLowersToTypeSwitch(t *testing.T) {
	out, err := backend.Transpile(sealedMatchSrc)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	for _, want := range []string{
		".(type) {", // a Go type-switch, not an Enum_Variant switch
		"case *Lit:", // concrete implementor case label
		"case *Neg:", //
	} {
		if !strings.Contains(out.Go, want) {
			t.Errorf("transpiled Go missing %q:\n%s", want, out.Go)
		}
	}
	if strings.Contains(out.Go, "Node_") {
		t.Errorf("sealed match must NOT lower through the enum §8.1 `Enum_Variant` path:\n%s", out.Go)
	}
}

// TestSealedMatchBehavesLikeTypeSwitch proves the lowered sealed match behaves
// identically to the equivalent hand-written `switch x := n.(type)`: the transpiled
// package is built into a throwaway `module goal` and run against a reference switch
// over the same implementor types.
func TestSealedMatchBehavesLikeTypeSwitch(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	out, err := backend.Transpile(sealedMatchSrc)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module goal\n\ngo 1.26\n")
	writeFile(t, filepath.Join(dir, "shape", "shape.go"), out.Go)
	writeFile(t, filepath.Join(dir, "shape", "behavior_test.go"), sealedMatchBehaviorTest)

	cmd := exec.Command("go", "test", "./shape/")
	cmd.Dir = dir
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("transpiled sealed match did not behave like the reference type-switch:\n%s", b)
	}
}

// sealedMatchBehaviorTest builds a small AST of Node values and asserts the
// transpiled `eval` (the lowered `match`) agrees with a reference `switch
// x := n.(type)` for every shape.
const sealedMatchBehaviorTest = `package shape

import "testing"

func reference(n Node) int {
	switch x := n.(type) {
	case *Lit:
		return x.Val
	case *Neg:
		return -reference(x.Inner)
	default:
		panic("unreachable")
	}
}

func TestEvalMatchesReferenceSwitch(t *testing.T) {
	cases := []Node{
		&Lit{Val: 7},
		&Neg{Inner: &Lit{Val: 7}},
		&Neg{Inner: &Neg{Inner: &Lit{Val: 3}}},
	}
	for _, c := range cases {
		if got, want := eval(c), reference(c); got != want {
			t.Errorf("eval(%T) = %d, reference switch = %d", c, got, want)
		}
	}
}
`
