package corpus

import (
	"strings"
	"testing"
)

// exhaustivenessDir is the corpus subtree of feature-02 (match) check cases — the
// exhaustiveness-related cases US-029 must pass through the AST checker.
const exhaustivenessDir = "testdata/check/02-match/"

// TestSemaExhaustiveRunner drives every exhaustiveness-related check case in the
// committed manifest through the AST-based [SemaCheck] via the [Checker] interface
// and the same inline // want markers the lexical checker is judged against. It
// fails loudly if the manifest yields no such cases, so a mis-generated manifest
// cannot masquerade as green.
func TestSemaExhaustiveRunner(t *testing.T) {
	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	ck := SemaCheck
	ran := 0
	for _, c := range m.Cases {
		if c.Kind != KindCheck || !strings.HasPrefix(c.Input, exhaustivenessDir) {
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
		t.Fatalf("manifest %q contains no %s check cases", manifestPath, exhaustivenessDir)
	}
}
