package byexample

import (
	"os"
	"strings"
	"testing"
)

// readDoc loads the repo's canonical by-example document for parsing.
func readDoc(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("../../docs/by-example.md")
	if err != nil {
		t.Fatalf("read by-example doc: %v", err)
	}
	return string(raw)
}

func TestParseFindsEveryFeature(t *testing.T) {
	doc, err := Parse(readDoc(t), "docs/by-example.md")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(doc.Categories) == 0 {
		t.Fatal("no categories parsed")
	}

	var total int
	for _, cat := range doc.Categories {
		if cat.Name == "" {
			t.Error("category with empty name")
		}
		for _, f := range cat.Features {
			total++
			if strings.TrimSpace(f.Source) == "" {
				t.Errorf("feature %q has empty Source", f.Title)
			}
			if f.SourceName == "" {
				t.Errorf("feature %q has empty SourceName", f.Title)
			}
			if f.Category != cat.Name {
				t.Errorf("feature %q Category=%q but is grouped under %q", f.Title, f.Category, cat.Name)
			}
			switch f.OutputKind {
			case "go", "test", "error", "doctest-failure":
			default:
				t.Errorf("feature %q has unexpected OutputKind %q", f.Title, f.OutputKind)
			}
		}
	}

	// The doc currently documents 13 name-tagged examples; the guide derives its
	// feature list from this count, so a drop is a regression worth catching.
	if total < 13 {
		t.Errorf("parsed %d features, want >= 13", total)
	}
}

// TestParseDoctestFailureBlock verifies that a feature whose locked output is a
// doctest-failure block (```testfail```) parses to the "doctest-failure" OutputKind
// with the failure body captured verbatim — distinct from a "Transpiles to" go block
// or a "Rejected with" error block.
func TestParseDoctestFailureBlock(t *testing.T) {
	const body = "--- FAIL: TestGreet (0.00s)\n    greet_test.go:1: got \"hi\", want \"hello\""
	doc := "# goal by Example\n\n" +
		"## 99. doctests: a failing doctest\n\n" +
		"A doctest whose expected output is wrong fails its generated test.\n\n" +
		"```goal name=greet.goal\n" +
		"func Greet() string { return \"hi\" }\n" +
		"/// >>> Greet()\n" +
		"/// hello\n" +
		"```\n\n" +
		"Fails with:\n\n" +
		"```testfail\n" +
		body + "\n" +
		"```\n"

	parsed, err := Parse(doc, "inline.md")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	var feats []Feature
	for _, cat := range parsed.Categories {
		feats = append(feats, cat.Features...)
	}
	if len(feats) != 1 {
		t.Fatalf("parsed %d features, want 1", len(feats))
	}
	f := feats[0]
	if f.OutputKind != "doctest-failure" {
		t.Errorf("OutputKind = %q, want %q", f.OutputKind, "doctest-failure")
	}
	if f.LockedExpected != body {
		t.Errorf("LockedExpected = %q, want %q", f.LockedExpected, body)
	}
}

func TestParseRejectsEmptyDoc(t *testing.T) {
	if _, err := Parse("# Nothing here\n\njust prose, no examples\n", "empty.md"); err == nil {
		t.Fatal("expected an error parsing a doc with no features, got nil")
	}
}
