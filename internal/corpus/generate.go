package corpus

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Generate walks the golden corpus rooted at root and returns a [Manifest]
// indexing every transpile pair and checker case. It is non-destructive: no
// source file is moved, renamed, or modified.
//
// Three locations are indexed:
//
//   - features/NN/examples/*.goal paired with a sibling *.go.expected, as
//     [KindTranspile] cases.
//   - top-level testdata/*.goal paired with a sibling *.go.expected, as
//     [KindTranspile] cases.
//   - testdata/check/**/*.goal (walked recursively), as [KindCheck] cases.
//     Checker expectations live inline as // want markers in the source, so the
//     emitted [Case] carries an empty Expected and [NormalizeNone].
//
// A transpile pair whose golden is a doctest sidecar (an emitted _test.go: it
// imports "testing" and contains a func Test...) additionally yields a
// [KindDoctest] case sharing the same Input and Expected. The transpile case is
// retained as-is, so the doctest case is purely additive — the doctest runner
// compares the emitted Output.Test sidecar against the same golden.
//
// Paths stored in the returned manifest are repo-root-relative and
// slash-separated for portability. Cases are sorted by Input so repeated
// generation over an unchanged corpus produces byte-identical output.
func Generate(root string) (Manifest, error) {
	var cases []Case

	// Transpile pairs: feature examples and top-level testdata.
	transpileGlobs := []string{
		filepath.Join(root, "features", "*", "examples", "*.goal"),
		filepath.Join(root, "testdata", "*.goal"),
	}
	for _, glob := range transpileGlobs {
		matches, err := filepath.Glob(glob)
		if err != nil {
			return Manifest{}, fmt.Errorf("corpus: globbing %q: %w", glob, err)
		}
		for _, in := range matches {
			expected := strings.TrimSuffix(in, ".goal") + ".go.expected"
			if !fileExists(expected) {
				// An unpaired .goal is not a transpile case.
				continue
			}
			rel := relSlash(root, in)
			expRel := relSlash(root, expected)
			cases = append(cases, Case{
				ID:        idFromRel(rel),
				Kind:      KindTranspile,
				Input:     rel,
				Expected:  expRel,
				Mode:      ModeFile,
				Normalize: NormalizeGofmt,
			})
			// When the golden is a doctest sidecar, additionally index a
			// doctest case so the doctest runner can compare Output.Test.
			if isDoctestSidecar(expected) {
				cases = append(cases, Case{
					ID:        idFromRel(rel) + "-doctest",
					Kind:      KindDoctest,
					Input:     rel,
					Expected:  expRel,
					Mode:      ModeFile,
					Normalize: NormalizeGofmt,
				})
			}
		}
	}

	// Check cases: every .goal under testdata/check, walked recursively.
	checkRoot := filepath.Join(root, "testdata", "check")
	err := filepath.WalkDir(checkRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".goal") {
			return nil
		}
		rel := relSlash(root, p)
		cases = append(cases, Case{
			ID:        idFromRel(rel),
			Kind:      KindCheck,
			Input:     rel,
			Expected:  "",
			Mode:      ModeFile,
			Normalize: NormalizeNone,
		})
		return nil
	})
	if err != nil {
		return Manifest{}, fmt.Errorf("corpus: walking %q: %w", checkRoot, err)
	}

	sort.Slice(cases, func(i, j int) bool { return cases[i].Input < cases[j].Input })
	return Manifest{Cases: cases}, nil
}

// fileExists reports whether p names an existing regular file.
func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.Mode().IsRegular()
}

// isDoctestSidecar reports whether the golden at expectedPath is an emitted
// doctest sidecar (a generated _test.go) rather than a main-output golden. A
// sidecar imports the "testing" package and declares at least one test
// function, which deterministically distinguishes the doctest examples (whose
// golden is the emitted _test.go) from ordinary transpile goldens — including
// sources that merely contain /// doctests but whose golden is the main output.
func isDoctestSidecar(expectedPath string) bool {
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		return false
	}
	s := string(data)
	return strings.Contains(s, `"testing"`) && strings.Contains(s, "func Test")
}

// relSlash returns p relative to root with forward slashes. If p is not under
// root, the cleaned slash form of p is returned.
func relSlash(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return filepath.ToSlash(p)
	}
	return filepath.ToSlash(rel)
}

// idFromRel derives a stable, unique case ID from a repo-relative path by
// dropping the .goal suffix and replacing path separators and dots with
// hyphens (e.g. "features/01-enums/examples/status.goal" ->
// "features-01-enums-examples-status").
func idFromRel(rel string) string {
	base := strings.TrimSuffix(rel, ".goal")
	base = path.Clean(base)
	repl := strings.NewReplacer("/", "-", ".", "-", " ", "-")
	return repl.Replace(base)
}
