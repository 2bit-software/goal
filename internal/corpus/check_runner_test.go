package corpus

import (
	"testing"

	"goal/internal/check"
)

// TestCheckRunner runs every check case in the committed manifest against the
// current lexical checker through the [Checker] interface and asserts all pass.
// It fails loudly if the manifest yields no check cases, so an empty or
// mis-generated manifest cannot masquerade as green.
func TestCheckRunner(t *testing.T) {
	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	ck := CheckerFunc(check.Analyze)
	ran := 0
	for _, c := range m.Cases {
		if c.Kind != KindCheck {
			continue
		}
		ran++
		c := c
		t.Run(c.ID, func(t *testing.T) {
			if err := RunCheck(repoRoot, c, ck); err != nil {
				t.Error(err)
			}
		})
	}

	if ran == 0 {
		t.Fatalf("manifest %q contains no check cases", manifestPath)
	}
}
