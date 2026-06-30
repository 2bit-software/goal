package corpus

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"goal/internal/compiler/parser"
)

// astParser wraps the AST front-end's parser.ParseFile as a [Parser].
var astParser = ParserFunc(func(src string) error {
	_, err := parser.ParseFile(src)
	return err
})

// TestParseGate is the US-024 gate: every unique .goal input referenced by the
// committed corpus manifest (file-mode inputs and every file of each
// package-mode case) MUST parse through the AST front-end with zero errors. The
// test fails loudly, listing every input that does not parse together with its
// error, so a parser regression names exactly which corpus inputs broke. It
// t.Fatalf's if the manifest yields no inputs, so an empty or mis-generated
// manifest cannot masquerade as green.
func TestParseGate(t *testing.T) {
	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	// Collect the unique .goal inputs across all cases, deduped by path
	// (doctest cases share an Input with their transpile twin), sorted for a
	// deterministic, diffable failure listing.
	seen := map[string]bool{}
	var inputs []string
	for _, c := range m.Cases {
		for _, in := range CaseInputs(c) {
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
		if err := astParser.Parse(string(src)); err != nil {
			failures = append(failures, in+": "+err.Error())
		}
	}

	if len(failures) > 0 {
		t.Errorf("%d of %d corpus inputs failed to parse:\n%s",
			len(failures), len(inputs), strings.Join(failures, "\n"))
	}
}
