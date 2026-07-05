package main

import (
	"bytes"
	"strings"
	"testing"
)

// The typed depth stage builds its go/types Config with a source importer
// (importer.ForCompiler(fset, "source", nil)), so it resolves a sibling goal
// package from that sibling's colocated generated .go. A call into a sibling
// with a wrong-typed argument therefore surfaces as a located
// `[go-type-error]`. With the old importer.Default() the sibling import failed
// to resolve and the cross-package call went unchecked (reported as `ok`), so
// this is the regression guard for the importer swap.
func TestCheckSurfacesCrossPackageGoTypeError(t *testing.T) {
	const mathx = `package mathx

func Double(n int) int {
	return n * 2
}
`
	const mainSrc = `package main

import (
	"fmt"

	"demo/mathx"
)

func main() {
	fmt.Println(mathx.Double("not an int"))
}
`
	dir := goalModule(t, map[string]string{
		"mathx/mathx.goal": mathx,
		"main.goal":        mainSrc,
	})

	// The source importer resolves a sibling from its colocated generated .go,
	// so emit the tree in place first (mirrors the committed generated Go that
	// `goal check ./internal` relies on). Run from the module root so the
	// relative emit target lands beside each package's source.
	t.Chdir(dir)
	var eb bytes.Buffer
	if err := run([]string{"build", "--emit=.", "."}, &bytes.Buffer{}, &eb); err != nil {
		t.Fatalf("emit failed: %v\n%s", err, eb.String())
	}

	var out, errOut bytes.Buffer
	err := run([]string{"check", "."}, &out, &errOut)
	if err == nil {
		t.Fatalf("expected check to fail on the cross-package type error\nstdout: %s\nstderr: %s", out.String(), errOut.String())
	}
	got := errOut.String()
	if !strings.Contains(got, "[go-type-error]") {
		t.Errorf("cross-package type error not surfaced with the [go-type-error] code:\n%s", got)
	}
	// The mismatched argument is located at the call site in main.goal.
	if !strings.Contains(got, "main.goal:") {
		t.Errorf("cross-package diagnostic not located in main.goal:\n%s", got)
	}
	if !strings.Contains(got, "mathx.Double") {
		t.Errorf("diagnostic does not name the sibling callee mathx.Double:\n%s", got)
	}
}
