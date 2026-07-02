package corpus

// US-020: foreign enum/sealed facts must be read from a dependency's .goal SOURCE
// even when a generated .go sits beside it, so match-exhaustiveness is checked
// against the current source (not a possibly-stale generated reconstruction); and
// individual foreign-file parse failures must be surfaced as warnings rather than
// swallowed by a bare continue.
//
// This is a handwritten Go test in the Go-only corpus infra, NOT a testdata/check
// fixture: a multi-package layout under testdata/check would be indexed per-file by
// the recursive corpus walker and mis-run. It drives sema.AnalyzePackageInDirWith
// directly with a fake DirResolver pointed at the fixture's dependency dir, so no
// Go toolchain resolution is needed.

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"goal/internal/sema"
)

// resolverTo returns a DirResolver that maps exactly importPath to depDir.
func resolverTo(importPath, depDir string) sema.DirResolver {
	return func(p, _ string) (string, error) {
		if p == importPath {
			return depDir, nil
		}
		return "", fmt.Errorf("unexpected import %q", p)
	}
}

// TestForeignEnumPrefersGoalSourceOverStaleGo: the dependency has both color.goal
// (Red, Green, Blue) and a stale color.go (Red, Green only). A consumer matching
// only Red and Green must be flagged non-exhaustive, naming the Blue variant only
// the .goal source declares.
func TestForeignEnumPrefersGoalSourceOverStaleGo(t *testing.T) {
	const importPath = "goal/internal/corpus/testdata/staledep/dep"
	depDir, err := filepath.Abs(filepath.Join("testdata", "staledep", "dep"))
	if err != nil {
		t.Fatal(err)
	}
	const consumerSrc = `package consumer

import "goal/internal/corpus/testdata/staledep/dep"

func describe(c dep.Color) string {
	label := match c {
		dep.Color.Red => "r"
		dep.Color.Green => "g"
	}
	return label
}
`
	perFile, _, err := sema.AnalyzePackageInDirWith([]string{consumerSrc}, filepath.Dir(depDir), resolverTo(importPath, depDir))
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	found := false
	for _, diags := range perFile {
		for _, d := range diags {
			if d.Code == "non-exhaustive-match" && strings.Contains(d.Message, "Blue") {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("expected a non-exhaustive-match diagnostic naming Blue (foreign enum must be read from .goal source, not the stale .go); got %+v", perFile)
	}
}

// TestForeignParseFailureSurfacedAsWarning: the dependency's color.goal is
// malformed. Enrichment must degrade to the valid .go facts (no crash, no empty
// enum) AND surface the .goal parse failure as a non-fatal warning naming the file,
// rather than swallowing it with a bare continue.
func TestForeignParseFailureSurfacedAsWarning(t *testing.T) {
	const importPath = "goal/internal/corpus/testdata/staledep_broken/dep"
	depDir, err := filepath.Abs(filepath.Join("testdata", "staledep_broken", "dep"))
	if err != nil {
		t.Fatal(err)
	}
	const consumerSrc = `package consumer

import "goal/internal/corpus/testdata/staledep_broken/dep"

func use(c dep.Color) dep.Color {
	return c
}
`
	_, ferrs, err := sema.AnalyzePackageInDirWith([]string{consumerSrc}, filepath.Dir(depDir), resolverTo(importPath, depDir))
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	found := false
	for _, e := range ferrs {
		if strings.Contains(e.Error(), "color.goal") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a surfaced warning naming color.goal (parse failure must not be swallowed); got %v", ferrs)
	}
}
