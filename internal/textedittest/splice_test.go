package textedittest

import (
	"reflect"
	"testing"

	"goal/internal/textedit"
)

// TestSpliceDropsOverlappingReplacement locks the conflict behavior: given two
// replacements whose spans overlap, Splice applies the first (by stable
// (Start, End) order) and returns the second in dropped rather than silently
// discarding it.
func TestSpliceDropsOverlappingReplacement(t *testing.T) {
	src := "abcdefgh"
	// Both start at offset 2; [2,5) is the shorter span and sorts first under the
	// (Start, End) tie-break, so it wins. [2,7) overlaps it and is dropped.
	reps := []textedit.Replacement{
		{Start: 2, End: 7, Text: "LONG"},
		{Start: 2, End: 5, Text: "X"},
	}
	out, dropped := textedit.Splice(src, 0, len(src), reps)

	if want := "abXfgh"; out != want {
		t.Fatalf("out = %q, want %q", out, want)
	}
	wantDropped := []textedit.Replacement{{Start: 2, End: 7, Text: "LONG"}}
	if !reflect.DeepEqual(dropped, wantDropped) {
		t.Fatalf("dropped = %+v, want %+v", dropped, wantDropped)
	}
}

// TestSpliceEqualStartDeterministic proves equal-Start conflicts resolve the
// same way regardless of input order — the stable (Start, End) sort always lets
// the shorter span win and drops the longer one.
func TestSpliceEqualStartDeterministic(t *testing.T) {
	src := "abcdefgh"
	shortFirst := []textedit.Replacement{
		{Start: 2, End: 5, Text: "X"},
		{Start: 2, End: 7, Text: "LONG"},
	}
	longFirst := []textedit.Replacement{
		{Start: 2, End: 7, Text: "LONG"},
		{Start: 2, End: 5, Text: "X"},
	}
	out1, dropped1 := textedit.Splice(src, 0, len(src), shortFirst)
	out2, dropped2 := textedit.Splice(src, 0, len(src), longFirst)

	if out1 != out2 {
		t.Fatalf("output not deterministic: %q vs %q", out1, out2)
	}
	if !reflect.DeepEqual(dropped1, dropped2) {
		t.Fatalf("dropped not deterministic: %+v vs %+v", dropped1, dropped2)
	}
	if want := "abXfgh"; out1 != want {
		t.Fatalf("out = %q, want %q", out1, want)
	}
	if len(dropped1) != 1 || dropped1[0].End != 7 {
		t.Fatalf("expected the [2,7) span dropped, got %+v", dropped1)
	}
}

// TestSpliceNoConflict is the control: non-overlapping replacements all apply
// and dropped is empty.
func TestSpliceNoConflict(t *testing.T) {
	src := "abcdefgh"
	reps := []textedit.Replacement{
		{Start: 5, End: 6, Text: "Y"},
		{Start: 1, End: 2, Text: "X"},
	}
	out, dropped := textedit.Splice(src, 0, len(src), reps)
	if want := "aXcdeYgh"; out != want {
		t.Fatalf("out = %q, want %q", out, want)
	}
	if len(dropped) != 0 {
		t.Fatalf("dropped = %+v, want empty", dropped)
	}
}
