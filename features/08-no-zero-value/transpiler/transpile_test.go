package main

import (
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExamples transpiles every examples/*.goal and asserts the output equals the
// matching *.go.expected after gofmt-normalizing both sides (so trivial whitespace
// differences do not matter). The pass is the definition of "the transpiler works."
func TestExamples(t *testing.T) {
	dir := filepath.Join("..", "examples")
	inputs, err := filepath.Glob(filepath.Join(dir, "*.goal"))
	if err != nil {
		t.Fatalf("glob examples: %v", err)
	}
	if len(inputs) == 0 {
		t.Fatalf("no examples found in %s", dir)
	}

	for _, in := range inputs {
		name := strings.TrimSuffix(filepath.Base(in), ".goal")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(in)
			if err != nil {
				t.Fatalf("read %s: %v", in, err)
			}
			got, err := transpile(string(src))
			if err != nil {
				t.Fatalf("transpile %s: %v", in, err)
			}

			wantPath := strings.TrimSuffix(in, ".goal") + ".go.expected"
			wantRaw, err := os.ReadFile(wantPath)
			if err != nil {
				t.Fatalf("read %s: %v", wantPath, err)
			}
			want := mustFormat(t, wantRaw)
			gotFmt := mustFormat(t, []byte(got))

			if gotFmt != want {
				t.Errorf("output mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, gotFmt, want)
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
