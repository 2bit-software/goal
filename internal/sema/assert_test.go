package sema

import (
	"strings"
	"testing"

	"goal/internal/parser"
)

// analyzeAssert parses src, resolves it, and returns the assert static-fold
// diagnostics.
func analyzeAssert(t *testing.T, src string) []Diagnostic {
	t.Helper()
	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return CheckAssert(file, Resolve(file))
}

func TestAssertAlwaysFalseLiteral(t *testing.T) {
	const src = `package p

func boom() {
	assert false
}
`
	d := analyzeAssert(t, src)
	if len(d) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(d), d)
	}
	if d[0].Severity != Error || d[0].Code != "assert-always-false" {
		t.Errorf("want Error assert-always-false, got %v %q", d[0].Severity, d[0].Code)
	}
	if !strings.Contains(d[0].Message, "statically false") {
		t.Errorf("message mismatch: %q", d[0].Message)
	}
}

func TestAssertAlwaysFalseCompare(t *testing.T) {
	const src = `package p

func bad() {
	assert 2 > 3
	assert 5 <= 4
	assert 1 == 2
	assert 7 != 7
}
`
	d := analyzeAssert(t, src)
	if len(d) != 4 {
		t.Fatalf("want 4 always-false diagnostics, got %d: %+v", len(d), d)
	}
	for _, x := range d {
		if x.Severity != Error || x.Code != "assert-always-false" {
			t.Errorf("want Error assert-always-false, got %v %q", x.Severity, x.Code)
		}
	}
}

func TestAssertAlwaysTrueIsWarning(t *testing.T) {
	const src = `package p

func taut() {
	assert true
	assert 3 < 4
	assert 5 >= 5
}
`
	d := analyzeAssert(t, src)
	if len(d) != 3 {
		t.Fatalf("want 3 always-true diagnostics, got %d: %+v", len(d), d)
	}
	for _, x := range d {
		if x.Severity != Warning || x.Code != "assert-always-true" {
			t.Errorf("want Warning assert-always-true, got %v %q", x.Severity, x.Code)
		}
		if !strings.Contains(x.Message, "always true") {
			t.Errorf("message mismatch: %q", x.Message)
		}
	}
}

func TestAssertRuntimeUndecidableIsClean(t *testing.T) {
	const src = `package p

func guard(n int) {
	assert n > 0
	assert balance >= amount
}
`
	if d := analyzeAssert(t, src); len(d) != 0 {
		t.Fatalf("an assert over a variable is runtime-checked and must draw nothing, got: %+v", d)
	}
}
