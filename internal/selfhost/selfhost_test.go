package selfhost_test

import (
	"strings"
	"testing"

	"goal/internal/project"
	"goal/internal/selfhost"
)

// TestGateFailsOnNonCompilingTranspile proves the gate is a real gate: a package
// that transpiles to non-compiling Go (here, an int-returning func with a bare
// return) must make BuildTranspiled fail. Without this, a green gate would be
// meaningless.
func TestGateFailsOnNonCompilingTranspile(t *testing.T) {
	const brokenSrc = "package brokenpkg\n\nfunc f() int { return }\n"
	pkg := &project.Package{
		// Dir "." is inside the module, so the front-end's import resolver finds
		// the enclosing go.mod (the package has no imports either way).
		Dir:   ".",
		Name:  "brokenpkg",
		Files: []project.File{{Path: "broken.go", Name: "broken" + project.Ext, Src: brokenSrc}},
	}
	err := selfhost.BuildTranspiled(map[string]*project.Package{"internal/brokenpkg": pkg})
	if err == nil {
		t.Fatal("expected the gate to fail on non-compiling transpiled Go, got nil")
	}
	if !strings.Contains(err.Error(), "brokenpkg") && !strings.Contains(err.Error(), "build") {
		t.Fatalf("expected a build/transpile failure mentioning the package, got: %v", err)
	}
}
