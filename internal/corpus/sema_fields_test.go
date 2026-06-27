package corpus

import (
	"strings"
	"testing"
)

// fieldCompletenessDir is the corpus subtree of feature-08 (no-zero-value) check
// cases — the field-completeness cases US-030 must pass through the AST checker.
const fieldCompletenessDir = "testdata/check/08-no-zero-value/"

// TestSemaFieldsRunner drives every field-completeness check case in the committed
// manifest through the AST-based [SemaCheck] via the [Checker] interface and the same
// inline // want markers the lexical checker is judged against. It fails loudly if the
// manifest yields no such cases, so a mis-generated manifest cannot masquerade as
// green.
func TestSemaFieldsRunner(t *testing.T) {
	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	ck := CheckerFunc(SemaCheck)
	ran := 0
	for _, c := range m.Cases {
		if c.Kind != KindCheck || !strings.HasPrefix(c.Input, fieldCompletenessDir) {
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
		t.Fatalf("manifest %q contains no %s check cases", manifestPath, fieldCompletenessDir)
	}
}
