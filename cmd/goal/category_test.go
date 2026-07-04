package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"goal/internal/guide"
)

// TestCategoryListsAllFeatures checks `goal category` prints one line per feature with a
// usage hint, covering the whole registry.
func TestCategoryListsAllFeatures(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run([]string{"category"}, &out, &errOut); err != nil {
		t.Fatalf("run category: %v\nstderr: %s", err, errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, "goal category <name>") {
		t.Error("list output omits the usage hint")
	}
	for _, c := range guide.Categories() {
		if !strings.Contains(got, c.Name) {
			t.Errorf("list output omits category %q", c.Name)
		}
	}
}

// TestCategoryDetailIsScoped checks a single feature renders in full without leaking another
// feature's example, and includes that feature's diagnostics.
func TestCategoryDetailIsScoped(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"category", "enums"}, &out, io.Discard); err != nil {
		t.Fatalf("run category enums: %v", err)
	}
	got := out.String()
	if !strings.HasPrefix(got, "## Enums") {
		t.Errorf("enums detail should start with its heading, got:\n%.60s", got)
	}
	if !strings.Contains(got, "[unknown-variant]") {
		t.Error("enums detail omits its diagnostics")
	}
	// No other feature's top-level heading should appear.
	for _, other := range []string{"## Match", "## implements", "## Doctests"} {
		if strings.Contains(got, other) {
			t.Errorf("enums detail leaked another feature's section %q", other)
		}
	}
}

// TestCategoryResultVariants checks the two Result features are addressable separately.
func TestCategoryResultVariants(t *testing.T) {
	var open, closed bytes.Buffer
	if err := run([]string{"category", "result"}, &open, io.Discard); err != nil {
		t.Fatalf("run category result: %v", err)
	}
	if err := run([]string{"category", "result-closed"}, &closed, io.Discard); err != nil {
		t.Fatalf("run category result-closed: %v", err)
	}
	if !strings.Contains(open.String(), "Result (open-E)") {
		t.Error("`result` did not render the open-E feature")
	}
	if !strings.Contains(closed.String(), "Result (closed-E)") {
		t.Error("`result-closed` did not render the closed-E feature")
	}
}

// TestCategoryNoDiagnosticFeatures checks features with no checker diagnostics still render
// successfully and exit zero.
func TestCategoryNoDiagnosticFeatures(t *testing.T) {
	for _, name := range []string{"option", "doctests"} {
		var out bytes.Buffer
		if err := run([]string{"category", name}, &out, io.Discard); err != nil {
			t.Errorf("run category %s: %v", name, err)
		}
		if out.Len() == 0 {
			t.Errorf("category %s produced no output", name)
		}
	}
}

// TestCategoryErrors checks the failure modes: an unknown name is a non-zero error that names
// the valid set, and more than one argument is a usage error.
func TestCategoryErrors(t *testing.T) {
	var out, errOut bytes.Buffer
	err := run([]string{"category", "bogus"}, &out, &errOut)
	if err == nil {
		t.Fatal("expected an error for an unknown category")
	}
	if !strings.Contains(err.Error(), "enums") {
		t.Errorf("unknown-category error should list valid names, got: %v", err)
	}

	out.Reset()
	errOut.Reset()
	if err := run([]string{"category", "a", "b"}, &out, &errOut); err == nil {
		t.Fatal("expected a usage error for extra arguments")
	}
}
