package pipeline

import (
	"strings"
	"testing"

	"goal/internal/project"
)

// TestPackageForeignErrorOnlyQuestion proves the whole arity-aware chain through the real
// driver: DefaultResolver parses the imported Go package, records ext.Mkdir's single return,
// and the `?` pass emits the one-value error guard rather than the two-value `_, ` form.
func TestPackageForeignErrorOnlyQuestion(t *testing.T) {
	src := `package conv

import ext "goal/internal/pipeline/testdata/extpkg"

func run() Result[int, error] {
	ext.Mkdir("d")?
	return Result.Ok(0)
}
`
	pkg := &project.Package{
		Dir:   "testdata",
		Name:  "conv",
		Files: []project.File{{Name: "conv.goal", Path: "testdata/conv.goal", Src: src}},
	}
	out, err := TranspilePackage(pkg)
	if err != nil {
		t.Fatalf("TranspilePackage: %v", err)
	}
	var gen string
	for _, f := range out.Files {
		if f.Name == "conv.go" {
			gen = f.Go
		}
	}
	if !strings.Contains(gen, "if __goal_err := ext.Mkdir(") {
		t.Errorf("error-only foreign `?` did not lower to the one-value guard\n--- got ---\n%s", gen)
	}
	if strings.Contains(gen, "_, __goal_err := ext.Mkdir(") {
		t.Errorf("error-only foreign `?` wrongly kept the two-value guard\n--- got ---\n%s", gen)
	}
}

// TestErrorOnlyQuestionCompiles is the SC-001 proof: an error-only `?` lowers to Go that
// actually builds (no `assignment mismatch`). The callee is in-file so the case is
// self-contained and needs no foreign import.
func TestErrorOnlyQuestionCompiles(t *testing.T) {
	src := `package demo

func clean() error {
	return nil
}

func sync() Result[int, error] {
	clean()?
	return Result.Ok(1)
}
`
	pkg := &project.Package{
		Dir:   "demo",
		Name:  "demo",
		Files: []project.File{{Name: "demo.goal", Path: "demo/demo.goal", Src: src}},
	}
	out, err := TranspilePackage(pkg)
	if err != nil {
		t.Fatalf("TranspilePackage: %v", err)
	}
	assertPackageCompiles(t, out)
}

// TestSingleFileForeignQuestionFallback documents the package-mode-only limitation: a
// single-file transpile is foreign-blind, so an unresolved foreign `?` callee keeps today's
// two-value form rather than regressing to a wrong one-value guess.
func TestSingleFileForeignQuestionFallback(t *testing.T) {
	src := `package conv

import ext "example.com/ext"

func run() Result[int, error] {
	ext.Mkdir("d")?
	return Result.Ok(0)
}
`
	out, err := Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	if !strings.Contains(out.Go, "_, __goal_err := ext.Mkdir(") {
		t.Errorf("single-file unresolved foreign `?` should keep the two-value form\n--- got ---\n%s", out.Go)
	}
}

// TestBindingErrorOnlyCalleeDiagnostic checks that binding a value from an error-only callee
// is reported as a goal-level error instead of silently emitting non-compiling Go.
func TestBindingErrorOnlyCalleeDiagnostic(t *testing.T) {
	src := `package demo

func clean() error {
	return nil
}

func sync() Result[int, error] {
	x := clean()?
	return Result.Ok(x)
}
`
	if _, err := Transpile(src); err == nil {
		t.Fatal("expected a diagnostic for binding a value from an error-only callee, got none")
	} else if !strings.Contains(err.Error(), "returns 1 value") {
		t.Errorf("diagnostic should explain the arity mismatch, got: %v", err)
	}
}
