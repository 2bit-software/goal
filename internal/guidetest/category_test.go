package guidetest

import (
	"bytes"
	"strings"
	"testing"

	"goal"
	"goal/internal/byexample"
	"goal/internal/guide"
)

// primaryFeatures returns the by-example features that represent a whole language feature —
// those whose anchor begins with a digit (the numbered "NN-…" headings), as opposed to the
// "Rejecting …" and composition sub-sections. These are exactly what `goal category` covers.
func primaryFeatures(t *testing.T) []byexample.Feature {
	t.Helper()
	raw, err := goal.Docs.ReadFile("docs/by-example.md")
	if err != nil {
		t.Fatalf("read by-example: %v", err)
	}
	doc, err := byexample.Parse(string(raw), "docs/by-example.md")
	if err != nil {
		t.Fatalf("parse by-example: %v", err)
	}
	var out []byexample.Feature
	for _, cat := range doc.Categories {
		for _, f := range cat.Features {
			if f.Anchor != "" && f.Anchor[0] >= '0' && f.Anchor[0] <= '9' {
				out = append(out, f)
			}
		}
	}
	return out
}

// TestCategoryCoverage checks every primary language feature is reachable through exactly
// one category, using only the exported API plus an independent parse of the doc.
func TestCategoryCoverage(t *testing.T) {
	features := primaryFeatures(t)
	names := guide.CategoryNames()
	if len(features) != len(names) {
		t.Fatalf("category count %d != primary feature count %d", len(names), len(features))
	}

	rendered := make(map[string]string, len(names))
	for _, name := range names {
		var b bytes.Buffer
		if err := guide.RenderCategory(&b, name); err != nil {
			t.Fatalf("RenderCategory(%q): %v", name, err)
		}
		if !strings.HasPrefix(b.String(), "## ") {
			t.Errorf("RenderCategory(%q) should start with a `## ` heading, got:\n%.60s", name, b.String())
		}
		rendered[name] = b.String()
	}

	for _, f := range features {
		hits := 0
		for _, body := range rendered {
			if strings.Contains(body, "## "+f.Title+"\n") {
				hits++
			}
		}
		if hits != 1 {
			t.Errorf("feature %q (anchor %s) is rendered by %d categories; want exactly 1", f.Title, f.Anchor, hits)
		}
	}
}

// TestCategoriesListShape checks the list view mirrors the name set in order and that the
// four core features are all valid categories.
func TestCategoriesListShape(t *testing.T) {
	infos := guide.Categories()
	names := guide.CategoryNames()
	if len(infos) != len(names) {
		t.Fatalf("Categories() len %d != CategoryNames() len %d", len(infos), len(names))
	}
	for i := range infos {
		if infos[i].Name != names[i] {
			t.Errorf("row %d: Categories name %q != CategoryNames %q", i, infos[i].Name, names[i])
		}
		if strings.TrimSpace(infos[i].Description) == "" {
			t.Errorf("category %q has an empty description", infos[i].Name)
		}
	}

	nameSet := map[string]bool{}
	for _, n := range names {
		nameSet[n] = true
	}
	for _, core := range []string{"enums", "match", "doctests", "implements"} {
		if !nameSet[core] {
			t.Errorf("core feature %q is not a valid category", core)
		}
	}
}

// TestCategoryDiagnostics checks a diagnostics-bearing feature includes its codes and that
// features without diagnostics render cleanly without a diagnostics block.
func TestCategoryDiagnostics(t *testing.T) {
	var enums bytes.Buffer
	if err := guide.RenderCategory(&enums, "enums"); err != nil {
		t.Fatalf("RenderCategory(enums): %v", err)
	}
	if !strings.Contains(enums.String(), "[unknown-variant]") {
		t.Error("enums detail omits its `unknown-variant` diagnostic")
	}

	for _, name := range []string{"option", "doctests"} {
		var b bytes.Buffer
		if err := guide.RenderCategory(&b, name); err != nil {
			t.Fatalf("RenderCategory(%q): %v", name, err)
		}
		if strings.Contains(b.String(), "**Diagnostics:**") {
			t.Errorf("%q has no checker diagnostics but rendered a Diagnostics block", name)
		}
	}
}

// TestCategoryResultsAreDistinct guards the two Result features against collapsing onto one
// name.
func TestCategoryResultsAreDistinct(t *testing.T) {
	var open, closed bytes.Buffer
	if err := guide.RenderCategory(&open, "result"); err != nil {
		t.Fatalf("RenderCategory(result): %v", err)
	}
	if err := guide.RenderCategory(&closed, "result-closed"); err != nil {
		t.Fatalf("RenderCategory(result-closed): %v", err)
	}
	if !strings.Contains(open.String(), "Result (open-E)") {
		t.Errorf("`result` did not render the open-E feature:\n%.60s", open.String())
	}
	if !strings.Contains(closed.String(), "Result (closed-E)") {
		t.Errorf("`result-closed` did not render the closed-E feature:\n%.60s", closed.String())
	}
	if open.String() == closed.String() {
		t.Error("`result` and `result-closed` rendered identical content")
	}
}

// TestCategoryUnknownNames checks an unknown category is an error that names the valid set.
func TestCategoryUnknownNames(t *testing.T) {
	var b bytes.Buffer
	err := guide.RenderCategory(&b, "bogus")
	if err == nil {
		t.Fatal("expected an error for an unknown category")
	}
	if !strings.Contains(err.Error(), "enums") {
		t.Errorf("unknown-category error should list valid names, got: %v", err)
	}
}
