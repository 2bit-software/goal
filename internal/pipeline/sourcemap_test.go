package pipeline

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"goal/internal/project"
)

// TestLineDirectivesMapErrorsToGoal is the integration proof for U5: a type error in a
// passed-through function body must be reported by `go build` at the .goal file and the
// line of that statement, not at the generated Go.
func TestLineDirectivesMapErrorsToGoal(t *testing.T) {
	// Line 1: package; 3: enum; 7: func area; the bad assignment is on line 8. A plain
	// Go type error (string into a float64) sits in passed-through code.
	const src = `package demo

enum Shape {
    Circle { r: float64 }
}

func area(sh Shape) float64 {
    var bad float64 = "not a number"
    return bad
}
`
	pkg := &project.Package{
		Dir:  "demo",
		Name: "demo",
		Files: []project.File{
			{Path: "demo/shapes.goal", Name: "shapes.goal", Src: src},
		},
	}
	out, err := TranspilePackage(pkg)
	if err != nil {
		t.Fatalf("TranspilePackage: %v", err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, f := range out.Files {
		if err := os.WriteFile(filepath.Join(dir, f.Name), []byte(f.Go), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	b, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected a compile error, got none")
	}
	msg := string(b)
	// The directive must remap the error to the .goal source, at the offending line.
	if !strings.Contains(msg, "shapes.goal:8") {
		t.Errorf("error not mapped to shapes.goal:8\n%s", msg)
	}
	if strings.Contains(msg, "shapes.go:") {
		t.Errorf("error leaked the generated-Go position instead of the .goal\n%s", msg)
	}
}

func TestAddLineDirectivesAnchorsUserDecls(t *testing.T) {
	goalSrc := "package demo\n\nfunc f() int {\n\treturn 0\n}\n"
	genGo := "package demo\n\nfunc f() int {\n\treturn 0\n}\n"
	got := addLineDirectives(goalSrc, genGo, "f.goal", "f.go")
	// f is declared on line 3 of the source; a directive must anchor it there.
	if !strings.Contains(got, "//line f.goal:3\n") {
		t.Errorf("missing source anchor for func f:\n%s", got)
	}
}
