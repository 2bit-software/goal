package goalfmt

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"goal/internal/corpus"
)

// repo-root-relative paths, matching the depth used by internal/corpus's own tests
// (internal/goalfmt sits at the same depth as internal/corpus).
const (
	repoRoot     = "../.."
	manifestPath = "../../corpus/manifest.json"
)

// TestIdempotentOverCorpus is the US-045 gate: formatting is idempotent for every
// unique .goal input referenced by the committed corpus manifest. For each input it
// asserts Source succeeds (the corpus is all well-formed goal, so formatting must
// never error — AC-4) and that a second format is a byte-for-byte no-op
// (Source(Source(src)) == Source(src) — AC-1). It t.Fatalf's if the manifest yields
// no inputs so an empty/mis-generated manifest cannot masquerade as green.
func TestIdempotentOverCorpus(t *testing.T) {
	m, err := corpus.Load(manifestPath)
	if err != nil {
		t.Fatalf("corpus.Load(%q): %v", manifestPath, err)
	}

	seen := map[string]bool{}
	var inputs []string
	for _, c := range m.Cases {
		for _, in := range corpus.CaseInputs(c) {
			if !seen[in] {
				seen[in] = true
				inputs = append(inputs, in)
			}
		}
	}
	sort.Strings(inputs)
	if len(inputs) == 0 {
		t.Fatalf("manifest %q contains no .goal inputs", manifestPath)
	}

	var failures []string
	for _, in := range inputs {
		src, err := os.ReadFile(filepath.Join(repoRoot, in))
		if err != nil {
			failures = append(failures, in+": read error: "+err.Error())
			continue
		}
		once, err := Source(string(src))
		if err != nil {
			failures = append(failures, in+": Source error: "+err.Error())
			continue
		}
		twice, err := Source(once)
		if err != nil {
			failures = append(failures, in+": second Source error: "+err.Error())
			continue
		}
		if once != twice {
			failures = append(failures, in+": not idempotent (fmt(fmt(src)) != fmt(src))")
		}
	}

	if len(failures) > 0 {
		t.Errorf("%d of %d corpus inputs failed formatting:\n%s",
			len(failures), len(inputs), strings.Join(failures, "\n"))
	}
}
