package backend_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"goal/internal/backend"
	"goal/internal/project"
)

// TestCrossPackageEnumMatchLowers is the SEAM-CAP regression: a `match` over an
// enum DEFINED in an imported package must lower to a Go type-switch over the
// imported §8.1 sum encoding (`case pkg.Enum_Variant:`), exactly as a same-package
// enum match does. Before SEAM-CAP it failed with
// `unsupported statement-position match on ""` because matchQualifier returned ""
// for the SelectorExpr variant pattern and the imported enum was not resolved.
func TestCrossPackageEnumMatchLowers(t *testing.T) {
	out := transpileCrossPkgEnum(t)

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
		"switch l.(type) {",
		"case light.Light_On:",
		"case light.Light_Off:",
	} {
		if !strings.Contains(useGo, want) {
			t.Errorf("transpiled use.go missing %q:\n%s", want, useGo)
		}
	}
}

// TestCrossPackageEnumMatchBehavesLikeSwitch proves the lowered cross-package
// enum match behaves identically to the equivalent hand-written Go type-switch:
// the transpiled package, the foreign enum package, and a reference switch are
// built into a throwaway module and run, asserting they agree variant-for-variant.
func TestCrossPackageEnumMatchBehavesLikeSwitch(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	out := transpileCrossPkgEnum(t)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module goal\n\ngo 1.26\n")

	// The foreign enum package at its import-path tail, so the generated import
	// `goal/internal/backend/testdata/extenum` resolves inside the temp module.
	extSrc, err := os.ReadFile(filepath.FromSlash("testdata/extenum/extenum.go"))
	if err != nil {
		t.Fatalf("read foreign fixture: %v", err)
	}
	writeFile(t, filepath.Join(dir, "internal", "backend", "testdata", "extenum", "extenum.go"), string(extSrc))

	// The transpiled `use` package.
	for _, f := range out.Files {
		writeFile(t, filepath.Join(dir, "use", f.Name), f.Go)
	}

	// A behavioral test that compares the transpiled `label` against a reference
	// switch over the same imported variants — identical mapping is the proof.
	writeFile(t, filepath.Join(dir, "use", "behavior_test.go"), crossPkgBehaviorTest)

	cmd := exec.Command("go", "test", "./use/")
	cmd.Dir = dir
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("transpiled cross-package enum match did not behave like the reference switch:\n%s", b)
	}
}

// transpileCrossPkgEnum runs the AST package driver over the cross-pkg-enum
// fixture and returns its output, failing the test on any transpile error.
func transpileCrossPkgEnum(t *testing.T) (out struct {
	Files []struct {
		Name string
		Go   string
	}
}) {
	t.Helper()
	src, err := os.ReadFile(filepath.FromSlash("../../testdata/package/cross-pkg-enum/use.goal"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	abs, err := filepath.Abs(filepath.FromSlash("../../testdata/package/cross-pkg-enum"))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	pkg := &project.Package{
		Dir:   abs,
		Name:  "use",
		Files: []project.File{{Path: "use.goal", Name: "use.goal", Src: string(src)}},
	}
	po, err := backend.TranspilePackage(pkg)
	if err != nil {
		t.Fatalf("TranspilePackage: %v", err)
	}
	for _, f := range po.Files {
		out.Files = append(out.Files, struct {
			Name string
			Go   string
		}{f.Name, f.Go})
	}
	return out
}

// writeFile writes content to path, creating parent directories, failing the test
// on error.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// crossPkgBehaviorTest exercises the transpiled `label` for every variant and
// asserts it equals a hand-written reference switch over the same imported types.
const crossPkgBehaviorTest = `package use

import (
	"testing"

	light "goal/internal/backend/testdata/extenum"
)

func reference(l light.Light) string {
	switch l.(type) {
	case light.Light_On:
		return "on"
	case light.Light_Off:
		return "off"
	default:
		panic("unreachable")
	}
}

func TestLabelMatchesReferenceSwitch(t *testing.T) {
	cases := []light.Light{light.Light_On{}, light.Light_Off{}}
	for _, c := range cases {
		if got, want := label(c), reference(c); got != want {
			t.Errorf("label(%T) = %q, reference switch = %q", c, got, want)
		}
	}
}
`
