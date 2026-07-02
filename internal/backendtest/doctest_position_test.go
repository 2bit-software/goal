package backendtest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"goal/internal/backend"
	"goal/internal/pipeline"
	"goal/internal/project"
)

// TestDoctestFailureNamesGoalPosition proves US-014: a failing package-mode
// doctest reports the .goal file and the source line of the `>>>` example in its
// `go test` output, instead of only the generated sidecar position.
//
// The doctest deliberately expects the wrong value, so the generated test fails
// at runtime; the run is compiled and executed in a throwaway module so the real
// `go test` output can be inspected. The `>>>` example is intentionally NOT the
// first line of its doc-comment run, which exercises the per-line numbering
// (a run-relative offset would report the wrong line).
func TestDoctestFailureNamesGoalPosition(t *testing.T) {
	// Line 1: package. Lines 3-6: the doc run. The `>>>` is on line 5.
	const src = `package example

/// answer returns a constant.
/// Example:
/// >>> answer()
/// 999
func answer() int {
	return 42
}
`
	const wantFile = "example.goal"
	const wantLine = 5 // the ">>>" line

	pkg := &project.Package{
		Dir:  ".",
		Name: "example",
		Files: []project.File{
			{Path: wantFile, Name: wantFile, Src: src},
		},
	}

	out, err := backend.TranspilePackage(pkg)
	if err != nil {
		t.Fatalf("TranspilePackage failed: %v", err)
	}
	if len(out.Tests) == 0 {
		t.Fatalf("no doctest sidecar produced")
	}

	tmp := t.TempDir()
	files := map[string]string{
		"go.mod": "module example\n\ngo 1.26\n",
	}
	for _, f := range out.Files {
		files[f.Name] = f.Go
	}
	for _, f := range out.Tests {
		files[f.Name] = f.Go
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte(body), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected the failing doctest to fail `go test`, but it passed\n%s", combined)
	}

	want := wantFile + ":" + strconv.Itoa(wantLine) + ":"
	if !strings.Contains(string(combined), want) {
		t.Errorf("doctest failure output does not name the .goal position %q\n--- go test output ---\n%s\n--- sidecar ---\n%s",
			want, combined, sidecarText(out.Tests))
	}
}

// sidecarText joins the generated doctest sidecar sources for diagnostics.
func sidecarText(tests []pipeline.GoFile) string {
	var b strings.Builder
	for _, f := range tests {
		b.WriteString(f.Go)
	}
	return b.String()
}
