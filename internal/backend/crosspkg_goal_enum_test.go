package backend_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"goal/internal/backend"
	"goal/internal/pipeline"
	"goal/internal/project"
)

// TestCrossPackageGoalEnumMatchLowers is the SEAM-CAP-2 regression: a `match` over — and
// a bare construction of — an enum DEFINED IN A SIBLING .goal PACKAGE (no generated .go)
// must lower, exercising the real per-package `goal build` topology. SEAM-CAP only handled
// the .go-defining-package case; here the defining package is .goal source, so the importer
// reads the enum via the new goalForeignDecls path. Before the fix the match failed
// (`unsupported expression *ast.MatchExpr`) and the bare construction lowered verbatim.
func TestCrossPackageGoalEnumMatchLowers(t *testing.T) {
	out := transpileGoalPkg(t, "testdata/goalenum/use", "use")

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
		"switch m.(type) {",                // cross-package match lowered to a type-switch
		"case mood.Mood_Happy:",            // over the imported §8.1 variant structs
		"case mood.Mood_Sad:",              //
		"mood.Mood(mood.Mood_Happy{})",     // bare cross-package construction in §8.1 form
	} {
		if !strings.Contains(useGo, want) {
			t.Errorf("transpiled use.go missing %q:\n%s", want, useGo)
		}
	}
}

// TestCrossPackageGoalEnumBehavesLikeSwitch proves the lowered cross-.goal-package enum
// match behaves identically to the equivalent hand-written Go type-switch: BOTH the
// defining package and the consumer are transpiled per-package (the real topology), built
// into a throwaway module, and run against a reference switch over the same variants.
func TestCrossPackageGoalEnumBehavesLikeSwitch(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	useOut := transpileGoalPkg(t, "testdata/goalenum/use", "use")
	moodOut := transpileGoalPkg(t, "testdata/goalenum/mood", "mood")

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module goal\n\ngo 1.26\n")

	// The transpiled defining package at its import-path tail, so the generated import
	// `goal/internal/backend/testdata/goalenum/mood` resolves inside the temp module.
	for _, f := range moodOut.Files {
		writeFile(t, filepath.Join(dir, "internal", "backend", "testdata", "goalenum", "mood", f.Name), f.Go)
	}
	for _, f := range useOut.Files {
		writeFile(t, filepath.Join(dir, "use", f.Name), f.Go)
	}
	writeFile(t, filepath.Join(dir, "use", "behavior_test.go"), crossPkgGoalBehaviorTest)

	cmd := exec.Command("go", "test", "./use/")
	cmd.Dir = dir
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("transpiled cross-.goal-package enum did not behave like the reference switch:\n%s", b)
	}
}

// transpileGoalPkg runs the AST package driver over every .goal file in relDir (resolved
// relative to this test's package) and returns the package output, failing on any error.
// The package Dir is the absolute fixture path so the foreign-import resolver finds the
// enclosing module's go.mod and resolves in-module imports to their sibling .goal dirs.
func transpileGoalPkg(t *testing.T, relDir, name string) pipeline.PackageOutput {
	t.Helper()
	abs, err := filepath.Abs(filepath.FromSlash(relDir))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		t.Fatalf("readdir %s: %v", abs, err)
	}
	var files []project.File
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".goal") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(abs, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		files = append(files, project.File{Path: e.Name(), Name: e.Name(), Src: string(src)})
	}
	if len(files) == 0 {
		t.Fatalf("no .goal files in %s", abs)
	}
	pkg := &project.Package{Dir: abs, Name: name, Files: files}
	out, err := backend.TranspilePackage(pkg)
	if err != nil {
		t.Fatalf("TranspilePackage(%s): %v", name, err)
	}
	return out
}

// crossPkgGoalBehaviorTest exercises the transpiled `label` (cross-package match) and
// `pick` (bare cross-package construction) for every variant and asserts they agree with
// a hand-written reference switch over the same imported §8.1 variant structs.
const crossPkgGoalBehaviorTest = `package use

import (
	"testing"

	mood "goal/internal/backend/testdata/goalenum/mood"
)

func reference(m mood.Mood) string {
	switch m.(type) {
	case mood.Mood_Happy:
		return "happy"
	case mood.Mood_Sad:
		return "sad"
	default:
		panic("unreachable")
	}
}

func TestLabelMatchesReferenceSwitch(t *testing.T) {
	cases := []mood.Mood{mood.Mood_Happy{}, mood.Mood_Sad{}}
	for _, c := range cases {
		if got, want := label(c), reference(c); got != want {
			t.Errorf("label(%T) = %q, reference switch = %q", c, got, want)
		}
	}
	if got := pick(); got != (mood.Mood(mood.Mood_Happy{})) {
		t.Errorf("pick() = %#v, want mood.Mood_Happy{}", got)
	}
}
`
