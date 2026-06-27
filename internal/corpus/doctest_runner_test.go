package corpus

import (
	"testing"

	"goal/internal/pipeline"
)

// TestDoctestRunner runs every doctest case in the committed manifest against
// the current pipeline front-end through the [Transpiler] interface and asserts
// all pass. It fails loudly if the manifest yields no doctest cases, so an empty
// or mis-generated manifest cannot masquerade as green.
func TestDoctestRunner(t *testing.T) {
	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	tp := TranspilerFunc(pipeline.Transpile)
	ran := 0
	for _, c := range m.Cases {
		if c.Kind != KindDoctest {
			continue
		}
		ran++
		c := c
		t.Run(c.ID, func(t *testing.T) {
			if err := RunDoctest(repoRoot, c, tp); err != nil {
				t.Error(err)
			}
		})
	}

	if ran == 0 {
		t.Fatalf("manifest %q contains no doctest cases", manifestPath)
	}
}
