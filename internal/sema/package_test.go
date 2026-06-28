package sema

import (
	"path/filepath"
	"testing"
)

// hasDiag reports whether diags contains a diagnostic of the given severity and code.
func hasDiag(diags []Diagnostic, sev Severity, code string) bool {
	for _, d := range diags {
		if d.Severity == sev && d.Code == code {
			return true
		}
	}
	return false
}

// TestAnalyzePackageInDirCrossFileExhaustiveness proves the package driver merges
// facts across files (FR-1) and returns one diagnostic list per file in input
// order (FR-2): an enum declared in file A makes a non-exhaustive `match` over it
// in file B a real error — a finding that a single-file check cannot produce.
func TestAnalyzePackageInDirCrossFileExhaustiveness(t *testing.T) {
	enumFile := `package demo

enum Shape {
    Circle { r: float64 }
    Square { s: float64 }
}
`
	matchFile := `package demo

func describe(sh Shape) string {
    return match sh {
        Shape.Circle(c) => "circle"
    }
}
`
	out, err := AnalyzePackageInDir([]string{enumFile, matchFile}, ".")
	if err != nil {
		t.Fatalf("AnalyzePackageInDir: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("want diagnostics for 2 files, got %d", len(out))
	}
	if hasDiag(out[0], Error, "non-exhaustive-match") {
		t.Error("the enum-only file (index 0) should carry no match error")
	}
	if !hasDiag(out[1], Error, "non-exhaustive-match") {
		t.Errorf("cross-file exhaustiveness not caught in file index 1; diags: %+v", out[1])
	}
}

// TestAnalyzePackageInDirForeignEnrichedDeriveFinding proves the driver folds in
// imported-package facts (FR-3): a `derive func` whose source is the foreign
// ext.Outer can only be proven incomplete once EnrichForeign loads ext.Outer's
// field set. With enrichment, the local target's `Extra` field has no source on
// ext.Outer, so the totality check fires `unsourced-field` (an Error). The
// control run, where the import fails to resolve, instead defers to the
// `unresolved-derive-type` Warning — so the Error genuinely depends on US-001
// foreign enrichment, not on cross-file resolution alone.
func TestAnalyzePackageInDirForeignEnrichedDeriveFinding(t *testing.T) {
	// Target is declared in a SIBLING file so the finding also exercises the
	// cross-file merge; the derive (importing the foreign package) lives in another.
	targetFile := `package consumer

type Target struct {
    ID string
    Extra string
}
`
	deriveFile := `package consumer

import ext "example.com/ext"

derive func make(o *ext.Outer) Target
`
	srcs := []string{targetFile, deriveFile}

	dir, err := filepath.Abs(filepath.Join("testdata", "extpkg"))
	if err != nil {
		t.Fatal(err)
	}
	resolve := func(importPath, _ string) (string, error) {
		if importPath != "example.com/ext" {
			t.Fatalf("unexpected import path %q", importPath)
		}
		return dir, nil
	}

	// Enriched: ext.Outer resolves, Target.Extra is unsourced -> Error.
	out, ferrs, err := AnalyzePackageInDirWith(srcs, ".", resolve)
	if err != nil {
		t.Fatalf("AnalyzePackageInDirWith: %v", err)
	}
	if len(ferrs) != 0 {
		t.Fatalf("unexpected enrichment errors: %v", ferrs)
	}
	if len(out) != 2 {
		t.Fatalf("want diagnostics for 2 files, got %d", len(out))
	}
	if !hasDiag(out[1], Error, "unsourced-field") {
		t.Errorf("foreign-enriched derive did not produce unsourced-field Error; diags: %+v", out[1])
	}
	if hasDiag(out[1], Warning, "unresolved-derive-type") {
		t.Errorf("with enrichment the source type should resolve, not defer; diags: %+v", out[1])
	}

	// Control: the import fails to resolve, so ext.Outer is never loaded and the
	// SAME derive can only defer with a Warning — proving the Error above depends
	// on foreign enrichment.
	boom := func(string, string) (string, error) { return "", errSentinel("nope") }
	ctrl, ctrlErrs, err := AnalyzePackageInDirWith(srcs, ".", boom)
	if err != nil {
		t.Fatalf("AnalyzePackageInDirWith (control): %v", err)
	}
	if len(ctrlErrs) != 1 {
		t.Fatalf("expected 1 non-fatal enrichment error in control, got %v", ctrlErrs)
	}
	if hasDiag(ctrl[1], Error, "unsourced-field") {
		t.Error("without enrichment there should be no unsourced-field Error (source type is unknown)")
	}
	if !hasDiag(ctrl[1], Warning, "unresolved-derive-type") {
		t.Errorf("without enrichment the derive should defer to a Warning; diags: %+v", ctrl[1])
	}
}

// TestAnalyzePackageInDirParseErrorReturned proves a parse failure in any file is
// surfaced as the driver's error result, not swallowed.
func TestAnalyzePackageInDirParseErrorReturned(t *testing.T) {
	good := "package p\n"
	bad := "package p\nfunc ( {\n"
	if _, err := AnalyzePackageInDir([]string{good, bad}, "."); err == nil {
		t.Fatal("expected a parse error for a malformed source file, got nil")
	}
}
