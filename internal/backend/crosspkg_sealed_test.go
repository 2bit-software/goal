package backend_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCrossPackageGoalSealedMatchLowers is the SEAM-CAP-3c lowering regression: a `match`
// over a SEALED INTERFACE DEFINED IN A SIBLING .goal PACKAGE (no generated .go) lowers to a
// Go type-switch with concrete qualified `case *shape.T:` labels, exercising the real
// per-package `goal build` topology. The implementor set is read from the sibling's .goal
// source via the extended goalForeignDecls path; before the fix the match's exhaustiveness
// deferred (`unresolved-match-sealed`) because the cross-package implementor set was unknown.
func TestCrossPackageGoalSealedMatchLowers(t *testing.T) {
	out := transpileGoalPkg(t, "testdata/goalsealed/use", "use")

	var useGo string
	for _, f := range out.Files {
		if f.Name == "use.go" {
			useGo = f.Go
		}
	}
	if useGo == "" {
		t.Fatal("no use.go in transpiled output")
	}
	for _, want := range []string{
		".(type) {",        // cross-package sealed match lowered to a Go type-switch
		"case *shape.Lit:", // over the imported concrete implementor types
		"case *shape.Neg:", //
	} {
		if !strings.Contains(useGo, want) {
			t.Errorf("transpiled use.go missing %q:\n%s", want, useGo)
		}
	}
	if strings.Contains(useGo, "Node_") {
		t.Errorf("sealed match must NOT lower through the enum §8.1 `Enum_Variant` path:\n%s", useGo)
	}
}

// TestCrossPackageGoalSealedBehavesLikeSwitch proves the lowered cross-.goal-package sealed
// match behaves identically to the equivalent hand-written `switch x := n.(type)`: BOTH the
// defining package and the consumer are transpiled per-package (the real topology), built
// into a throwaway module, and run against a reference type-switch over the same implementors.
func TestCrossPackageGoalSealedBehavesLikeSwitch(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	useOut := transpileGoalPkg(t, "testdata/goalsealed/use", "use")
	shapeOut := transpileGoalPkg(t, "testdata/goalsealed/shape", "shape")

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module goal\n\ngo 1.26\n")

	// The transpiled defining package at its import-path tail, so the generated import
	// `goal/internal/backend/testdata/goalsealed/shape` resolves inside the temp module.
	for _, f := range shapeOut.Files {
		writeFile(t, filepath.Join(dir, "internal", "backend", "testdata", "goalsealed", "shape", f.Name), f.Go)
	}
	for _, f := range useOut.Files {
		writeFile(t, filepath.Join(dir, "use", f.Name), f.Go)
	}
	writeFile(t, filepath.Join(dir, "use", "behavior_test.go"), crossPkgGoalSealedBehaviorTest)

	cmd := exec.Command("go", "test", "./use/")
	cmd.Dir = dir
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("transpiled cross-.goal-package sealed match did not behave like the reference switch:\n%s", b)
	}
}

// crossPkgGoalSealedBehaviorTest exercises the transpiled `eval` (cross-package sealed
// match) for a small AST of shape.Node values and asserts it agrees with a hand-written
// reference `switch x := n.(type)` over the same imported implementor types.
const crossPkgGoalSealedBehaviorTest = `package use

import (
	"testing"

	shape "goal/internal/backend/testdata/goalsealed/shape"
)

func reference(n shape.Node) int {
	switch x := n.(type) {
	case *shape.Lit:
		return x.Val
	case *shape.Neg:
		return -reference(x.Inner)
	default:
		panic("unreachable")
	}
}

func TestEvalMatchesReferenceSwitch(t *testing.T) {
	cases := []shape.Node{
		&shape.Lit{Val: 7},
		&shape.Neg{Inner: &shape.Lit{Val: 7}},
		&shape.Neg{Inner: &shape.Neg{Inner: &shape.Lit{Val: 3}}},
	}
	for _, c := range cases {
		if got, want := eval(c), reference(c); got != want {
			t.Errorf("eval(%T) = %d, reference switch = %d", c, got, want)
		}
	}
}
`
