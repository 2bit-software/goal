package pipeline

import (
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTestdata transpiles every testdata/*.goal multi-feature program and asserts the
// output equals the matching *.go.expected after gofmt-normalizing both sides. These
// programs combine features no single reference transpiler handles (Result + Option +
// `?`), so the pass is the proof that the front-end composes.
func TestTestdata(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata")
	inputs, err := filepath.Glob(filepath.Join(dir, "*.goal"))
	if err != nil {
		t.Fatalf("glob testdata: %v", err)
	}
	if len(inputs) == 0 {
		t.Fatalf("no testdata found in %s", dir)
	}

	for _, in := range inputs {
		name := strings.TrimSuffix(filepath.Base(in), ".goal")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(in)
			if err != nil {
				t.Fatalf("read %s: %v", in, err)
			}
			got, err := Transpile(string(src))
			if err != nil {
				t.Fatalf("transpile %s: %v", in, err)
			}

			wantPath := strings.TrimSuffix(in, ".goal") + ".go.expected"
			wantRaw, err := os.ReadFile(wantPath)
			if err != nil {
				t.Fatalf("read %s: %v", wantPath, err)
			}
			want := mustFormat(t, wantRaw)
			gotFmt := mustFormat(t, []byte(got.Go))
			if gotFmt != want {
				t.Errorf("output mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, gotFmt, want)
			}
		})
	}
}

// TestSingleFeatureRegression runs the per-feature reference examples that the
// unified pipeline now subsumes (Result, Option, `?`) and asserts each still lowers
// to its original expected Go. These guard against the consolidation regressing any
// single-feature behavior the standalone transpilers had.
func TestSingleFeatureRegression(t *testing.T) {
	dirs := []string{
		filepath.Join("..", "..", "features", "01-enums", "examples"),
		filepath.Join("..", "..", "features", "02-match", "examples"),
		filepath.Join("..", "..", "features", "03-result", "examples"),
		filepath.Join("..", "..", "features", "04-option", "examples"),
		filepath.Join("..", "..", "features", "05-question-prop", "examples"),
		filepath.Join("..", "..", "features", "06-error-e", "examples"),
		filepath.Join("..", "..", "features", "07-implements", "examples"),
		filepath.Join("..", "..", "features", "08-no-zero-value", "examples"),
		filepath.Join("..", "..", "features", "09-pure", "examples"),
		filepath.Join("..", "..", "features", "10-assert", "examples"),
	}
	for _, dir := range dirs {
		inputs, err := filepath.Glob(filepath.Join(dir, "*.goal"))
		if err != nil {
			t.Fatalf("glob %s: %v", dir, err)
		}
		for _, in := range inputs {
			name := filepath.Base(filepath.Dir(filepath.Dir(in))) + "/" + strings.TrimSuffix(filepath.Base(in), ".goal")
			t.Run(name, func(t *testing.T) {
				src, err := os.ReadFile(in)
				if err != nil {
					t.Fatalf("read %s: %v", in, err)
				}
				got, err := Transpile(string(src))
				if err != nil {
					t.Fatalf("transpile %s: %v", in, err)
				}
				wantRaw, err := os.ReadFile(strings.TrimSuffix(in, ".goal") + ".go.expected")
				if err != nil {
					t.Fatalf("read expected: %v", err)
				}
				if mustFormat(t, []byte(got.Go)) != mustFormat(t, wantRaw) {
					t.Errorf("output mismatch for %s\n--- got ---\n%s", name, got.Go)
				}
			})
		}
	}
}

// TestDoctests runs the feature-11 reference examples through the pipeline and
// asserts the doctest side output (Output.Test) equals the expected `_test.go`. The
// main Go output is the source verbatim, since `///` is a valid Go comment.
func TestDoctests(t *testing.T) {
	dir := filepath.Join("..", "..", "features", "11-doctests", "examples")
	inputs, err := filepath.Glob(filepath.Join(dir, "*.goal"))
	if err != nil {
		t.Fatalf("glob %s: %v", dir, err)
	}
	for _, in := range inputs {
		name := strings.TrimSuffix(filepath.Base(in), ".goal")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(in)
			if err != nil {
				t.Fatalf("read %s: %v", in, err)
			}
			got, err := Transpile(string(src))
			if err != nil {
				t.Fatalf("transpile %s: %v", in, err)
			}
			if got.Test == "" {
				t.Fatalf("expected doctest output for %s, got none", name)
			}
			wantRaw, err := os.ReadFile(strings.TrimSuffix(in, ".goal") + ".go.expected")
			if err != nil {
				t.Fatalf("read expected: %v", err)
			}
			if mustFormat(t, []byte(got.Test)) != mustFormat(t, wantRaw) {
				t.Errorf("doctest mismatch for %s\n--- got ---\n%s", name, got.Test)
			}
		})
	}
}

func mustFormat(t *testing.T, src []byte) string {
	t.Helper()
	out, err := format.Source(src)
	if err != nil {
		t.Fatalf("gofmt failed: %v\n--- source ---\n%s", err, src)
	}
	return string(out)
}
