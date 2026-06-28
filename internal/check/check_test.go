package check_test

import (
	"testing"

	"goal/internal/check"
	"goal/internal/corpus"
)

// manifestPath is the committed corpus manifest, and repoRoot the repo root,
// both relative to this package directory (internal/check). They match the
// depth the corpus package itself uses.
const (
	manifestPath = "../../corpus/manifest.json"
	repoRoot     = "../.."
)

// TestCorpusCheck is the whole checker harness: it loads every check case from
// the committed corpus manifest and runs each through the shared corpus check
// runner, which matches diagnostics against the inline `// want` markers in the
// source. Cases live in the manifest, not in a path walk hardcoded here —
// adding one is regenerating the manifest, never editing this file. It fails
// loudly if the manifest yields no check cases, so an empty or mis-generated
// manifest cannot masquerade as green.
func TestCorpusCheck(t *testing.T) {
	m, err := corpus.Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	ck := corpus.CheckerFunc(check.Analyze)
	ran := 0
	for _, c := range m.Cases {
		if c.Kind != corpus.KindCheck {
			continue
		}
		ran++
		c := c
		t.Run(c.ID, func(t *testing.T) {
			if err := corpus.RunCheck(repoRoot, c, ck); err != nil {
				t.Error(err)
			}
		})
	}

	if ran == 0 {
		t.Fatalf("manifest %q contains no check cases", manifestPath)
	}
}

// TestRegistryRuns is a spine smoke test: every registered check runs without
// error on representative source, independent of whether any check is
// implemented yet.
func TestRegistryRuns(t *testing.T) {
	const src = `package p

enum Shape {
	Circle { r: float64 }
	Rect   { w: float64, h: float64 }
}
`
	if _, err := check.Analyze(src); err != nil {
		t.Fatalf("checker spine errored on valid source: %v", err)
	}
}
