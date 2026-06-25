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
			case "go", "test", "error":
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

func TestParseRejectsEmptyDoc(t *testing.T) {
	if _, err := Parse("# Nothing here\n\njust prose, no examples\n", "empty.md"); err == nil {
		t.Fatal("expected an error parsing a doc with no features, got nil")
	}
}
