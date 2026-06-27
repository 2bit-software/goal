package goalfmt

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"goal/internal/corpus"
)

// repo-root-relative paths, matching the depth used by internal/corpus's own tests
// (internal/goalfmt sits at the same depth as internal/corpus).
const (
	repoRoot     = "../.."
	manifestPath = "../../corpus/manifest.json"
)

// TestIdempotentOverCorpus is the US-045 gate: formatting is idempotent for every
// unique .goal input referenced by the committed corpus manifest. For each input it
// asserts Source succeeds (the corpus is all well-formed goal, so formatting must
// never error — AC-4) and that a second format is a byte-for-byte no-op
// (Source(Source(src)) == Source(src) — AC-1). It t.Fatalf's if the manifest yields
// no inputs so an empty/mis-generated manifest cannot masquerade as green.
func TestIdempotentOverCorpus(t *testing.T) {
	m, err := corpus.Load(manifestPath)
	if err != nil {
		t.Fatalf("corpus.Load(%q): %v", manifestPath, err)
	}

	seen := map[string]bool{}
	var inputs []string
	for _, c := range m.Cases {
		for _, in := range corpus.CaseInputs(c) {
			if !seen[in] {
				seen[in] = true
				inputs = append(inputs, in)
			}
		}
	}
	sort.Strings(inputs)
	if len(inputs) == 0 {
		t.Fatalf("manifest %q contains no .goal inputs", manifestPath)
	}

	var failures []string
	for _, in := range inputs {
		src, err := os.ReadFile(filepath.Join(repoRoot, in))
		if err != nil {
			failures = append(failures, in+": read error: "+err.Error())
			continue
		}
		once, err := Source(string(src))
		if err != nil {
			failures = append(failures, in+": Source error: "+err.Error())
			continue
		}
		twice, err := Source(once)
		if err != nil {
			failures = append(failures, in+": second Source error: "+err.Error())
			continue
		}
		if once != twice {
			failures = append(failures, in+": not idempotent (fmt(fmt(src)) != fmt(src))")
		}
	}

	if len(failures) > 0 {
		t.Errorf("%d of %d corpus inputs failed formatting:\n%s",
			len(failures), len(inputs), strings.Join(failures, "\n"))
	}
}

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
