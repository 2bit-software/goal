package guide

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"goal"
	"goal/internal/byexample"
)

// TestLiveLowerings asserts every by-example feature still transpiles through the live
// pipeline, so the guide can never show a broken or stale lowering. This is the core of
// the "generated from the real transpiler" guarantee.
func TestLiveLowerings(t *testing.T) {
	raw, err := goal.Docs.ReadFile("docs/by-example.md")
	if err != nil {
		t.Fatalf("read by-example: %v", err)
	}
	doc, err := byexample.Parse(string(raw), "docs/by-example.md")
	if err != nil {
		t.Fatalf("parse by-example: %v", err)
	}
	var n int
	for _, cat := range doc.Categories {
		for _, f := range cat.Features {
			n++
			if _, _, err := lower(f); err != nil {
				t.Errorf("feature %q: %v", f.Title, err)
			}
		}
	}
	if n == 0 {
		t.Fatal("no features exercised")
	}
}

// codeLiteralRe matches a kebab-case diagnostic-code string literal. In internal/check
// and internal/typecheck the only hyphenated lowercase string literals are diagnostic
// codes, so scanning for these yields exactly the set of codes the checker can emit.
var codeLiteralRe = regexp.MustCompile(`"([a-z]+(?:-[a-z]+)+)"`)

// TestDiagnosticCatalogMatchesSource holds the documented catalog to the codes the
// checker actually emits: it scans the checker source for code literals and fails on
// any code documented-but-absent or emitted-but-undocumented. Adding a diagnostic
// without cataloguing it (or removing one) breaks this test.
func TestDiagnosticCatalogMatchesSource(t *testing.T) {
	live := scanLiveCodes(t)
	documented := catalogCodes()

	for code := range live {
		if !documented[code] {
			t.Errorf("code %q is emitted by the checker but missing from diagnosticCatalog", code)
		}
	}
	for code := range documented {
		if !live[code] {
			t.Errorf("code %q is in diagnosticCatalog but no longer emitted by the checker", code)
		}
	}
}

// scanLiveCodes returns the diagnostic codes found in the non-test source of
// internal/check and internal/typecheck.
func scanLiveCodes(t *testing.T) map[string]bool {
	t.Helper()
	codes := map[string]bool{}
	for _, pkg := range []string{"../check", "../typecheck"} {
		entries, err := os.ReadDir(pkg)
		if err != nil {
			t.Fatalf("read %s: %v", pkg, err)
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(pkg, name))
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			for _, m := range codeLiteralRe.FindAllStringSubmatch(string(data), -1) {
				codes[m[1]] = true
			}
		}
	}
	if len(codes) == 0 {
		t.Fatal("scanned no diagnostic codes — scan path or regex is wrong")
	}
	return codes
}

// TestRenderFullGuideCoversEverySection checks the full guide includes each section's
// heading, so no section silently drops out of the assembly.
func TestRenderFullGuideCoversEverySection(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, "", testCommands()); err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := buf.String()
	for _, s := range sections {
		if !strings.Contains(out, "## "+s.title) {
			t.Errorf("full guide is missing section %q (%q)", s.key, s.title)
		}
	}
}

// TestRenderUnknownSection reports the valid section names on an unknown key.
func TestRenderUnknownSection(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, "nope", testCommands())
	if err == nil {
		t.Fatal("expected an error for an unknown section")
	}
	for _, key := range SectionKeys() {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error %q does not list valid section %q", err, key)
		}
	}
}

func testCommands() []Command {
	return []Command{{Name: "build", Summary: "build it", Usage: "goal build [path]"}}
}
