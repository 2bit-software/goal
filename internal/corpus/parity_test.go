package corpus

import (
	"os"
	"path/filepath"
	"testing"

	"goal/internal/check"
)

// finding is the comparable projection of a diagnostic used by the parity gate:
// (file, line, feature, code, severity). Message text is intentionally excluded —
// the lexical and AST checkers word their messages differently by design, so the
// gate compares which guarantee fires where, at what severity, not how it is
// phrased. Line is the 1-based source line (check.OffsetToPosition); severity is
// the lowercase label ("error"/"warning").
type finding struct {
	File     string
	Line     int
	Feature  string
	Code     string
	Severity string
}

// divergence is one finding produced by exactly one of the two checkers, tagged
// with the side that produced it. Every entry in knownDivergences MUST be backed
// by a documented note in DECISIONS.md ("US-003 — differential checker parity
// gate").
type divergence struct {
	finding
	Side string // "sema" or "legacy"
}

// knownDivergences is the DECISIONS.md-backed allowlist of accepted sema↔legacy
// disagreements over the check corpus. The parity gate subtracts these before
// requiring the two checkers' findings to be identical. It is an error for an
// entry here to NOT reproduce (a stale entry), so this list stays honest as the
// AST checker evolves. See DECISIONS.md "US-003 — differential checker parity
// gate" for the rationale of each entry.
var knownDivergences = []divergence{
	// Divergences 1–3 (improvement): the AST checker resolves the in-file derive
	// target and fires a real Error where the lexical checker swallows the trailing
	// // want comment into the target type name, fails to resolve it, and defers
	// with unresolved-derive-type (Warning).
	{finding{"testdata/check/12-derive-convert/fallible_in_total.goal", 24, "12-derive-convert", "fallible-in-total-derive", "error"}, "sema"},
	{finding{"testdata/check/12-derive-convert/fallible_in_total.goal", 24, "12-derive-convert", "unresolved-derive-type", "warning"}, "legacy"},
	{finding{"testdata/check/12-derive-convert/unbridged_field.goal", 19, "12-derive-convert", "unbridged-field", "error"}, "sema"},
	{finding{"testdata/check/12-derive-convert/unbridged_field.goal", 19, "12-derive-convert", "unresolved-derive-type", "warning"}, "legacy"},
	{finding{"testdata/check/12-derive-convert/unsourced_field.goal", 18, "12-derive-convert", "unsourced-field", "error"}, "sema"},
	{finding{"testdata/check/12-derive-convert/unsourced_field.goal", 18, "12-derive-convert", "unresolved-derive-type", "warning"}, "legacy"},

	// Divergence 4 (extra deferral): the AST checker surfaces a located
	// unresolved-err-value Warning (closedness deferred for a Result.Err on a bound
	// variable); the lexical checker emits nothing there. Both are non-rejecting.
	{finding{"testdata/check/06-error-e/defer_err_value.goal", 16, "06-error-e", "unresolved-err-value", "warning"}, "sema"},
}

// TestSemaLegacyParity is the differential parity gate (US-003). It runs both the
// AST checker (SemaCheck) and the legacy lexical checker (check.Analyze) over
// every KindCheck case in the committed manifest and compares findings by
// (file, line, feature, code, severity). The two must agree exactly, except for
// the DECISIONS.md-backed divergences in knownDivergences. The gate fails on:
//
//   - an empty check corpus (a mis-generated manifest cannot masquerade as green),
//   - any divergence not present in the allowlist (an undocumented disagreement),
//   - any allowlist entry that does not reproduce (a stale, rotting entry).
//
// Passing it is the safety proof that deleting internal/check (US-005) loses no
// guarantee the AST checker does not already provide.
func TestSemaLegacyParity(t *testing.T) {
	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	allowed := map[divergence]bool{}
	for _, d := range knownDivergences {
		allowed[d] = true
	}
	seen := map[divergence]bool{}

	ran := 0
	for _, c := range m.Cases {
		if c.Kind != KindCheck {
			continue
		}
		ran++

		srcBytes, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(c.Input)))
		if err != nil {
			t.Fatalf("case %q: reading input: %v", c.ID, err)
		}
		src := string(srcBytes)

		legacy, err := check.Analyze(src)
		if err != nil {
			t.Fatalf("case %q: legacy check.Analyze: %v", c.ID, err)
		}
		semad, err := SemaCheck(src)
		if err != nil {
			t.Fatalf("case %q: SemaCheck: %v", c.ID, err)
		}

		lcount := countFindings(c.Input, src, legacy)
		scount := countFindings(c.Input, src, semad)

		// Walk the union of finding keys; the multiset difference on each side is a
		// per-side divergence. Comparing counts (not just presence) catches a
		// duplicate one checker emits and the other does not.
		union := map[finding]bool{}
		for f := range lcount {
			union[f] = true
		}
		for f := range scount {
			union[f] = true
		}
		for f := range union {
			for i := 0; i < scount[f]-lcount[f]; i++ {
				d := divergence{f, "sema"}
				seen[d] = true
				if !allowed[d] {
					t.Errorf("undocumented divergence: SEMA-only finding %s:%d [%s/%s] %s — add to DECISIONS.md + knownDivergences or fix the checker",
						f.File, f.Line, f.Feature, f.Code, f.Severity)
				}
			}
			for i := 0; i < lcount[f]-scount[f]; i++ {
				d := divergence{f, "legacy"}
				seen[d] = true
				if !allowed[d] {
					t.Errorf("undocumented divergence: LEGACY-only finding %s:%d [%s/%s] %s — add to DECISIONS.md + knownDivergences or fix the checker",
						f.File, f.Line, f.Feature, f.Code, f.Severity)
				}
			}
		}
	}

	if ran == 0 {
		t.Fatalf("manifest %q contains no check cases", manifestPath)
	}

	for d := range allowed {
		if !seen[d] {
			t.Errorf("stale allowlist entry no longer reproduces: %s-only %s:%d [%s/%s] %s — remove it from knownDivergences and DECISIONS.md",
				d.Side, d.File, d.Line, d.Feature, d.Code, d.Severity)
		}
	}
}

// countFindings projects a checker's diagnostics for one case file into a
// finding-keyed multiset (count per (file, line, feature, code, severity)).
func countFindings(file, src string, diags []check.Diagnostic) map[finding]int {
	out := map[finding]int{}
	for _, d := range diags {
		out[finding{
			File:     file,
			Line:     check.OffsetToPosition(src, d.Pos).Line,
			Feature:  d.Feature,
			Code:     d.Code,
			Severity: d.Severity.String(),
		}]++
	}
	return out
}
