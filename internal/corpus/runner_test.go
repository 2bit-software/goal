package corpus

import (
	"testing"

	"goal/internal/pipeline"
)

// manifestPath is the committed corpus manifest, relative to this package
// directory (internal/corpus). repoRoot ("../..") is declared in
// generate_test.go in this same package.
const manifestPath = "../../corpus/manifest.json"

// TestTranspileRunner runs every transpile case in the committed manifest
// against the current pipeline front-end through the [Transpiler] interface and
// asserts all pass. It fails loudly if the manifest yields no transpile cases,
// so an empty or mis-generated manifest cannot masquerade as green.
func TestTranspileRunner(t *testing.T) {
	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	tp := TranspilerFunc(pipeline.Transpile)
	ran := 0
	for _, c := range m.Cases {
		if c.Kind != KindTranspile {
			continue
		}
		ran++
		c := c
		t.Run(c.ID, func(t *testing.T) {
			if err := RunTranspile(repoRoot, c, tp); err != nil {
				t.Error(err)
			}
		})
	}

	if ran == 0 {
		t.Fatalf("manifest %q contains no transpile cases", manifestPath)
	}
}
