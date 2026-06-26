package lsp

import (
	"testing"

	"goal/internal/check"
)

// A non-exhaustive match is the invalid program used across the LSP tests.
const nonExhaustiveSrc = `package p

enum Light {
	Red
	Green
}

func f(l Light) string {
	x := match l {
		Light.Red => "r"
	}
	return x
}
`

// A check finding's 1-based byte offset becomes a 0-based protocol range, with
// goal severities mapped to the protocol scale and the line-end as the span end.
func TestToLSPMapping(t *testing.T) {
	src := "line one\nsecond line here\n"
	// Offset 9 is the first byte of the second line.
	d := check.Diagnostic{Pos: 9, Severity: check.Error, Code: "demo", Message: "boom"}

	got := toLSP(src, nil, d)
	if got.Range.Start.Line != 1 || got.Range.Start.Character != 0 {
		t.Fatalf("start = %+v, want {1,0}", got.Range.Start)
	}
	if want := len("second line here"); got.Range.End.Character != want {
		t.Fatalf("end char = %d, want %d", got.Range.End.Character, want)
	}
	if got.Severity != 1 {
		t.Fatalf("severity = %d, want 1 (Error)", got.Severity)
	}
	if got.Source != "goal" || got.Code != "demo" || got.Message != "boom" {
		t.Fatalf("metadata mismatch: %+v", got)
	}
}

// When the finding's offset is a known token start, the range covers exactly that token; with
// no token-end known it falls back to the end of the line.
func TestToLSPRangeUsesTokenEnd(t *testing.T) {
	src := "alpha beta\n"
	d := check.Diagnostic{Pos: 0, Severity: check.Error, Code: "x"}

	got := toLSP(src, map[int]int{0: 5}, d) // "alpha" spans [0,5)
	if got.Range.End.Line != 0 || got.Range.End.Character != 5 {
		t.Fatalf("token-end range = %+v, want end {0,5}", got.Range.End)
	}
	fallback := toLSP(src, map[int]int{}, d) // offset not in map → line end
	if fallback.Range.End.Character != len("alpha beta") {
		t.Fatalf("fallback end char = %d, want %d", fallback.Range.End.Character, len("alpha beta"))
	}
}

// Warning-severity findings map to the protocol's Warning level.
func TestToLSPWarningSeverity(t *testing.T) {
	got := toLSP("abc\n", nil, check.Diagnostic{Pos: 0, Severity: check.Warning})
	if got.Severity != 2 {
		t.Fatalf("severity = %d, want 2 (Warning)", got.Severity)
	}
}

// The reused check surface rejects an invalid program, which is what the server
// turns into editor diagnostics.
func TestAnalyzeProducesDiagnostic(t *testing.T) {
	diags, err := check.Analyze(nonExhaustiveSrc)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(diags) == 0 {
		t.Fatal("expected at least one diagnostic for non-exhaustive match")
	}
}
