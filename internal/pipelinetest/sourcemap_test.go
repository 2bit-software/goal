package pipelinetest

import (
	"strings"
	"testing"

	"goal/internal/pipeline"
)

// TestSameNamedMethodsMapToOwnLine proves two methods that share a name but sit on
// different receivers each anchor to their own source line. Before declName folded
// the receiver into the mapping key, both collided on the bare method name and the
// second method silently inherited the first's //line, misreporting its position.
func TestSameNamedMethodsMapToOwnLine(t *testing.T) {
	src := "package p\n" + // 1
		"\n" + // 2
		"type A struct{}\n" + // 3
		"type B struct{}\n" + // 4
		"\n" + // 5
		"func (a A) Get() int { return 1 }\n" + // 6
		"\n" + // 7
		"func (b B) Get() int { return 2 }\n" // 8

	out, err := pipeline.AddLineDirectives(src, src, "p.goal", "p.go")
	if err != nil {
		t.Fatalf("AddLineDirectives: %v", err)
	}

	if got := directiveBefore(t, out, "func (a A) Get"); got != "//line p.goal:6" {
		t.Errorf("A.Get anchored to %q, want //line p.goal:6", got)
	}
	if got := directiveBefore(t, out, "func (b B) Get"); got != "//line p.goal:8" {
		t.Errorf("B.Get anchored to %q, want //line p.goal:8 (regression: shared the method-name key)", got)
	}
}

// directiveBefore returns the line immediately preceding the first line containing
// needle in out.
func directiveBefore(t *testing.T, out, needle string) string {
	t.Helper()
	lines := strings.Split(out, "\n")
	for i, ln := range lines {
		if strings.Contains(ln, needle) {
			if i == 0 {
				t.Fatalf("%q has no preceding line", needle)
			}
			return lines[i-1]
		}
	}
	t.Fatalf("%q not found in output", needle)
	return ""
}
