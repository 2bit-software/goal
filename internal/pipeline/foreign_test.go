package pipeline

import (
	"strings"
	"testing"

	"goal/internal/project"
)

// TestTranspilePackageForeignDerive exercises the whole foreign-type chain through the
// real package driver: DefaultResolver resolves the same-module import via go.mod, the
// imported Go package is parsed for its struct field set, and the `derive func` over the
// out-of-package `*ext.Outer` lowers to a field-by-field conversion. The package Dir is a
// real path under this module so the offline same-module resolver finds it.
func TestTranspilePackageForeignDerive(t *testing.T) {
	src := `package conv

import ext "goal/internal/pipeline/testdata/extpkg"

type Local struct {
	ID    string
	Count int
}

derive func make(o *ext.Outer) Local
`
	pkg := &project.Package{
		Dir:  "testdata",
		Name: "conv",
		Files: []project.File{
			{Name: "conv.goal", Path: "testdata/conv.goal", Src: src},
		},
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
	if gen == "" {
		t.Fatal("no conv.go in output")
	}
	for _, want := range []string{
		"func make(o *ext.Outer) Local",
		"out.ID = o.ID",
		"out.Count = o.Count",
	} {
		if !strings.Contains(gen, want) {
			t.Errorf("generated Go missing %q\n--- got ---\n%s", want, gen)
		}
	}
}
