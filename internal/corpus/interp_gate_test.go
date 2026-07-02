package corpus

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// interpGateSkips is the EXPLICIT skip list for the whole-corpus interpreter
// behavioral gate (TestInterpWholeCorpusBehavioralGate). It maps a doctest
// corpus case ID to the justification for excluding it from interpretation —
// e.g. the case uses a stdlib symbol the host-function bridge does not yet shim.
//
// It exists to forbid silent drops (REWRITE-ARCHITECTURE.md §7 Phase 5 / §6,
// "honest deferral over false completion"): a case the interpreter cannot run
// must be recorded here with a reason, never quietly omitted. The gate fails if
// any entry has a blank reason or names a case that is not a doctest case in the
// committed manifest.
//
// The blank-reason enforcement is exercised regardless by
// TestInterpGateSkipListRejectsBlankReason, so the mechanism is proven whatever
// the list contains.
var interpGateSkips = map[string]string{
	// The goscript interpreter does not yet evaluate func literals (evalExpr has
	// no *ast.FuncLit case) and cannot resolve a nameless closure's
	// Result-propagation signature (sigFor keys sema.FuncSignatures by name).
	// US-004's closure-lowering fix targets the Go backend; the compiled doctest
	// sidecar oracle (TestDoctestRunner) covers this case behaviorally. Adding
	// closure evaluation to the interpreter is deferred, recorded here rather
	// than silently dropped.
	"testdata-closure_result-doctest": "interpreter does not yet evaluate func literals (*ast.FuncLit); backend doctest sidecar oracle covers this case",
}

// TestInterpWholeCorpusBehavioralGate is US-027: the whole-corpus behavioral
// parity gate for the goscript interpreter. It runs every applicable corpus case
// — the doctest-kind subset, which the interpreter executes behaviorally in
// process via RunInterp (no Go toolchain) — through interpretation and reports
// zero behavioral failures. Each `///  >>>` example's evaluated runtime result
// must equal its documented expected value.
//
// This is the interpreter analogue of TestASTEngineWholeCorpusBehavioralGate:
// judged by behavior rather than Go spelling, the goscript interpreter is held
// to the SAME implementation-independent yardstick as the Go back-end across the
// whole applicable corpus (REWRITE-ARCHITECTURE.md §6). Any excluded case must be
// enumerated in interpGateSkips with a reason; a behavioral failure, an
// unjustified skip, a stale skip, or an empty/narrowed corpus all fail the gate
// rather than passing it silently.
func TestInterpWholeCorpusBehavioralGate(t *testing.T) {
	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}
	if len(m.Cases) == 0 {
		t.Fatalf("manifest %q is empty — a filtered or mis-generated manifest cannot pass as green", manifestPath)
	}

	// The skip list must be honest: every entry needs a recorded reason (FR-4)
	// and must name a real doctest case in the manifest, so a stale or
	// reason-less skip cannot quietly narrow the gate (FR-5).
	if bad := blankSkipReasons(interpGateSkips); len(bad) > 0 {
		t.Errorf("interpreter gate skip list has entries with no recorded reason: %v", bad)
	}
	doctestIDs := map[string]bool{}
	for _, c := range m.Cases {
		if c.Kind == KindDoctest {
			doctestIDs[c.ID] = true
		}
	}
	for id := range interpGateSkips {
		if !doctestIDs[id] {
			t.Errorf("interpreter gate skip list names %q, which is not a doctest case in the manifest", id)
		}
	}

	var ran, skipped int
	for _, c := range m.Cases {
		if c.Kind != KindDoctest {
			continue
		}
		if reason, ok := interpGateSkips[c.ID]; ok {
			t.Logf("skipping %q under interpretation: %s", c.ID, reason)
			skipped++
			continue
		}
		ran++
		c := c
		t.Run(c.ID, func(t *testing.T) {
			if err := RunInterp(repoRoot, c); err != nil {
				t.Error(err)
			}
		})
	}

	if ran == 0 {
		t.Fatalf("interpreter gate ran no doctest cases — an empty or mis-generated manifest cannot pass as green")
	}
	t.Logf("goscript interpreter: %d doctest case(s) green under interpretation, %d skipped", ran, skipped)
}

// TestInterpGateSkipListRejectsBlankReason proves the gate's skip-list
// enforcement (FR-4): blankSkipReasons returns exactly the IDs whose reason is
// blank or whitespace-only, and nothing for a fully justified list. This keeps
// the "fail on an unjustified skip" guarantee exercised even while the real
// interpGateSkips list ships empty.
func TestInterpGateSkipListRejectsBlankReason(t *testing.T) {
	bad := blankSkipReasons(map[string]string{
		"has-reason": "uses an unshimmed stdlib symbol",
		"no-reason":  "",
		"whitespace": "   \t ",
	})
	want := []string{"no-reason", "whitespace"}
	if !reflect.DeepEqual(bad, want) {
		t.Fatalf("blankSkipReasons = %v, want %v", bad, want)
	}

	if got := blankSkipReasons(map[string]string{"ok": "a justified reason"}); len(got) != 0 {
		t.Fatalf("a fully justified skip list must report no blank reasons, got %v", got)
	}
	if got := blankSkipReasons(map[string]string{}); len(got) != 0 {
		t.Fatalf("an empty skip list must report no blank reasons, got %v", got)
	}
}

// blankSkipReasons returns the sorted IDs in skips whose justification is blank
// (empty or whitespace-only). The whole-corpus gate uses it to refuse a skip-list
// entry that records no reason.
func blankSkipReasons(skips map[string]string) []string {
	var bad []string
	for id, reason := range skips {
		if strings.TrimSpace(reason) == "" {
			bad = append(bad, id)
		}
	}
	sort.Strings(bad)
	return bad
}
