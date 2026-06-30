package corpus

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"goal/internal/compiler/backend"
)

// updateGoldens, when set, makes TestUpdateGoldens rewrite every exact-tier
// golden (.go.expected) from the AST backend's output instead of skipping. It
// mirrors the parser package's -update-snapshots flag: the regeneration
// mechanism is durable and reproducible, but off by default so an ordinary test
// run never mutates the corpus.
var updateGoldens = flag.Bool("update-goldens", false, "regenerate .go.expected goldens from the AST backend")

// TestUpdateGoldens regenerates the checked-in .go.expected goldens for every
// file-mode transpile case from the AST backend (backend.Transpile), so the
// exact conformance tier measures the canonical AST front-end. For a case whose
// golden is an emitted doctest _test.go sidecar (isDoctestSidecar), it writes
// the gofmt-normalized doctest output (Output.Test); for every other case it
// writes the gofmt-normalized main output (Output.Go). The feature-11 doctest
// cases share their golden with their transpile twin, so iterating the file-mode
// transpile cases regenerates the doctest sidecars too.
//
// It is a no-op (skipped) unless -update-goldens is passed. Goldens are produced
// only from real backend output here — never hand-edited.
func TestUpdateGoldens(t *testing.T) {
	if !*updateGoldens {
		t.Skip("set -update-goldens to regenerate the corpus goldens")
	}

	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	tp := TranspilerFunc(backend.Transpile)
	updated := 0
	for _, c := range m.Cases {
		if c.Kind != KindTranspile || c.Mode != ModeFile {
			continue // package-mode cases have no exact golden (behavioral only)
		}

		srcPath := filepath.Join(repoRoot, filepath.FromSlash(c.Input))
		src, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("case %q: reading input: %v", c.ID, err)
		}

		out, err := tp.Transpile(string(src))
		if err != nil {
			t.Fatalf("case %q: transpile: %v", c.ID, err)
		}

		goldenPath := filepath.Join(repoRoot, filepath.FromSlash(c.Expected))
		// Classify the golden BEFORE overwriting it: a doctest-sidecar golden is
		// the emitted _test.go (Output.Test), every other golden the main output.
		raw := out.Go
		if isDoctestSidecar(goldenPath) {
			raw = out.Test
		}

		formatted, err := gofmtNormalize(raw)
		if err != nil {
			t.Fatalf("case %q: gofmt output: %v", c.ID, err)
		}
		if err := os.WriteFile(goldenPath, []byte(formatted), 0o644); err != nil {
			t.Fatalf("case %q: writing golden %s: %v", c.ID, goldenPath, err)
		}
		updated++
	}

	if updated == 0 {
		t.Fatalf("manifest %q contains no file-mode transpile cases", manifestPath)
	}
	t.Logf("regenerated %d goldens from the AST backend", updated)
}
