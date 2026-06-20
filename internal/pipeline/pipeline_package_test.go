package pipeline

import (
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"goal/internal/project"
)

// typesGoal declares the cross-file symbols; mathGoal references both — an enum from
// the sibling file (via match) and a closed-E error type from it (via Result[..,
// MathErr]). This is the package shape SPIKE-2 proved compiles with one shared prelude.
const typesGoal = `package demo

enum Shape {
    Circle { r: float64 }
    Square { s: float64 }
}

enum MathErr {
    DivZero
}
`

const mathGoal = `package demo

func area(sh Shape) float64 {
    return match sh {
        Shape.Circle(c) => 3.14159 * c.r * c.r
        Shape.Square(q) => q.s * q.s
    }
}

func half(x float64) Result[float64, MathErr] {
    if x == 0.0 {
        return Result.Err(MathErr.DivZero)
    }
    return Result.Ok(x / 2.0)
}
`

func demoPackage() *project.Package {
	return &project.Package{
		Dir:  "demo",
		Name: "demo",
		Files: []project.File{
			{Path: "demo/math.goal", Name: "math.goal", Src: mathGoal},
			{Path: "demo/types.goal", Name: "types.goal", Src: typesGoal},
		},
	}
}

func TestTranspilePackageCrossFile(t *testing.T) {
	out, err := TranspilePackage(demoPackage())
	if err != nil {
		t.Fatalf("TranspilePackage: %v", err)
	}

	files := map[string]string{}
	for _, f := range out.Files {
		if _, dup := files[f.Name]; dup {
			t.Fatalf("duplicate output file %s", f.Name)
		}
		files[f.Name] = f.Go
	}

	// Exactly the two sources plus one shared prelude.
	for _, want := range []string{"math.go", "types.go", "goal_prelude.go"} {
		if _, ok := files[want]; !ok {
			t.Errorf("missing generated file %s (got %v)", want, keys(files))
		}
	}
	if len(out.Files) != 3 {
		t.Errorf("got %d files, want 3: %v", len(out.Files), keys(files))
	}

	// The prelude appears once and carries the sum encoding.
	if !strings.Contains(files["goal_prelude.go"], "type Result[T, E any] interface") {
		t.Error("goal_prelude.go missing the closed-E sum encoding")
	}

	// math.go's match over the cross-file enum lowered to a real type switch, and its
	// closed-E Result lowered against MathErr (also cross-file).
	mg := files["math.go"]
	if !strings.Contains(mg, "switch") || !strings.Contains(mg, "Shape_Circle") {
		t.Error("cross-file enum match did not lower to a type switch")
	}
	if !strings.Contains(mg, "Ok[float64, MathErr]") {
		t.Error("closed-E Result against the cross-file error type did not lower")
	}

	// Every generated file is valid Go.
	for name, src := range files {
		if _, err := format.Source([]byte(src)); err != nil {
			t.Errorf("%s is not valid Go: %v", name, err)
		}
	}

	// The strongest proof: the whole package compiles together.
	assertPackageCompiles(t, out)
}

// assertPackageCompiles writes the package output into a throwaway module and runs
// `go build`, the real check that the cross-file lowering + single prelude cohere.
func assertPackageCompiles(t *testing.T, out PackageOutput) {
	t.Helper()
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
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated package failed to compile: %v\n%s", err, b)
	}
}

func keys(m map[string]string) []string {
	var k []string
	for name := range m {
		k = append(k, name)
	}
	return k
}
