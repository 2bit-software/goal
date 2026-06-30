package goalfmt

import (
	"strings"
	"testing"
)

// This file holds the self-contained (fixture-free, stdlib-only) goalfmt behavior
// tests. They are split out of format_test.go so the self-host port gate
// (internal/selfhost TestPortedGoalfmtPackage) can run them against the transpiled
// goal-built goalfmt inside its throwaway temp module — the corpus/manifest-backed
// TestIdempotentOverCorpus stays behind in format_test.go because it reads
// repo-relative fixtures absent from that module. Keep these tests dependency-light
// (strings + testing only) so they remain self-contained.

// TestPreservesComments asserts that every // line comment and /// doc comment in the
// input survives formatting (AC-2). The parser drops // comments, so a naive
// AST-printing formatter would lose them; goalfmt rewrites raw source lines, so they
// are preserved verbatim.
func TestPreservesComments(t *testing.T) {
	src := "" +
		"package sample\n" +
		"\n" +
		"// a leading line comment about Greeting\n" +
		"/// Greeting returns a greeting.\n" +
		"/// >>> Greeting()\n" +
		"/// \"hi\"\n" +
		"func Greeting() string {\n" +
		"        // an indented comment inside the body\n" +
		"        return \"hi\" // a trailing comment\n" +
		"}\n"

	out, err := Source(src)
	if err != nil {
		t.Fatalf("Source: %v", err)
	}

	comments := []string{
		"// a leading line comment about Greeting",
		"/// Greeting returns a greeting.",
		"/// >>> Greeting()",
		"// an indented comment inside the body",
		"// a trailing comment",
	}
	for _, c := range comments {
		if !strings.Contains(out, c) {
			t.Errorf("formatted output dropped comment %q\n--- output ---\n%s", c, out)
		}
	}

	// Idempotency on the sample too.
	out2, err := Source(out)
	if err != nil {
		t.Fatalf("second Source: %v", err)
	}
	if out != out2 {
		t.Errorf("sample not idempotent:\n--- once ---\n%s\n--- twice ---\n%s", out, out2)
	}
}

// TestReindentsAndIsStable checks that mis-indented (space-indented, ragged) source
// is normalized to tab indentation by nesting depth and is stable on re-run.
func TestReindentsAndIsStable(t *testing.T) {
	src := "" +
		"package p\n" +
		"func f() int {\n" +
		"  if true {\n" +
		"        return 1\n" +
		"  }\n" +
		"  return 0\n" +
		"}\n"

	want := "" +
		"package p\n" +
		"func f() int {\n" +
		"\tif true {\n" +
		"\t\treturn 1\n" +
		"\t}\n" +
		"\treturn 0\n" +
		"}\n"

	out, err := Source(src)
	if err != nil {
		t.Fatalf("Source: %v", err)
	}
	if out != want {
		t.Errorf("reindent mismatch:\n--- got ---\n%q\n--- want ---\n%q", out, want)
	}
	out2, err := Source(out)
	if err != nil {
		t.Fatalf("second Source: %v", err)
	}
	if out != out2 {
		t.Errorf("not idempotent:\n--- once ---\n%q\n--- twice ---\n%q", out, out2)
	}
}

// TestRejectsUnparseable asserts that malformed goal source is reported as an error
// and not silently reformatted (AC-3).
func TestRejectsUnparseable(t *testing.T) {
	if _, err := Source("package p\nfunc f( {\n"); err == nil {
		t.Fatal("Source accepted unparseable source; want a parse error")
	}
}
