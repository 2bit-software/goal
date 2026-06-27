package pipeline_test

import (
	"testing"

	"goal/internal/corpus"
	"goal/internal/pipeline"
)

// manifestPath is the committed corpus manifest, and repoRoot the repo root,
// both relative to this package directory (internal/pipeline). They match the
// depth the corpus package itself uses.
const (
	manifestPath = "../../corpus/manifest.json"
	repoRoot     = "../.."
)

// TestCorpusTranspile runs every transpile case in the committed corpus
// manifest through the current pipeline front-end via the shared corpus runner,
// which gofmt-normalizes both sides before comparing. The cases — the testdata
// multi-feature programs and the per-feature reference examples the unified
// pipeline subsumes — live in the manifest, not in a directory list hardcoded
// here. It fails loudly if the manifest yields no transpile cases, so an empty
// or mis-generated manifest cannot masquerade as green.
func TestCorpusTranspile(t *testing.T) {
	m, err := corpus.Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	tp := corpus.TranspilerFunc(pipeline.Transpile)
	ran := 0
	for _, c := range m.Cases {
		if c.Kind != corpus.KindTranspile || c.Mode != corpus.ModeFile {
			continue // package-mode cases run through the package runner
		}
		ran++
		c := c
		t.Run(c.ID, func(t *testing.T) {
			if err := corpus.RunTranspile(repoRoot, c, tp); err != nil {
				t.Error(err)
			}
		})
	}

	if ran == 0 {
		t.Fatalf("manifest %q contains no transpile cases", manifestPath)
	}
}

// TestCorpusDoctest runs every doctest case in the committed corpus manifest
// through the pipeline via the shared corpus doctest runner, which asserts the
// emitted `_test.go` sidecar (Output.Test) matches the golden after
// gofmt-normalizing both sides. It fails loudly if the manifest yields no
// doctest cases.
func TestCorpusDoctest(t *testing.T) {
	m, err := corpus.Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	tp := corpus.TranspilerFunc(pipeline.Transpile)
	ran := 0
	for _, c := range m.Cases {
		if c.Kind != corpus.KindDoctest {
			continue
		}
		ran++
		c := c
		t.Run(c.ID, func(t *testing.T) {
			if err := corpus.RunDoctest(repoRoot, c, tp); err != nil {
				t.Error(err)
			}
		})
	}

	if ran == 0 {
		t.Fatalf("manifest %q contains no doctest cases", manifestPath)
	}
}
