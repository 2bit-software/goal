package corpus

import (
	"strings"
	"testing"
)

// us031Dirs are the corpus subtrees whose check cases US-031 reimplements over the
// AST: must-use (03-result), closed-E `?` From-totality and Err closedness
// (06-error-e), and interface satisfaction (07-implements). The open-E `?` arity/refusal
// check (feature 05) has no dedicated fixture dir; its clean path rides inside the
// 03-result and 06-error-e cases.
var us031Dirs = []string{
	"testdata/check/03-result/",
	"testdata/check/06-error-e/",
	"testdata/check/07-implements/",
}

// TestSemaQuestionImplementsRunner drives every must-use, implements, and closed-E `?`
// check case in the committed manifest through the AST-based [SemaCheck] via the
// [Checker] interface and the same inline // want markers the lexical checker is judged
// against. It fails loudly if the manifest yields no such cases, so a mis-generated
// manifest cannot masquerade as green.
func TestSemaQuestionImplementsRunner(t *testing.T) {
	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	ck := CheckerFunc(SemaCheck)
	ran := 0
	for _, c := range m.Cases {
		if c.Kind != KindCheck || !hasAnyPrefix(c.Input, us031Dirs) {
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
		t.Fatalf("manifest %q contains no US-031 (03/06/07) check cases", manifestPath)
	}
}

// hasAnyPrefix reports whether s starts with any of the given prefixes.
func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
