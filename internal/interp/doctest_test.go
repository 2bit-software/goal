package interp

// These tests prove the interpreter-side doctest oracle (US-025 foundation):
// RunDoctests evaluates each `///  >>>` example through the shared AST + sema
// front-end and reports a loud, identified failure on a wrong result — never a
// silent pass. The corpus runner (internal/corpus.RunInterp) drives this so the
// behavioral tier judges the interpreter on the same yardstick as the Go backend.

import (
	"strings"
	"testing"
)

const doctestProgram = `package mathx

/// Adds two ints.
/// >>> add(2, 3)
/// 5
func add(a int, b int) int {
	return a + b
}

/// repeat concatenates s n times.
/// >>> repeat("ab", 2)
/// "abab"
func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
`

func TestRunDoctestsAllPass(t *testing.T) {
	failures, ran, err := RunDoctests(doctestProgram)
	if err != nil {
		t.Fatalf("RunDoctests: unexpected error: %v", err)
	}
	if ran != 2 {
		t.Errorf("ran = %d, want 2", ran)
	}
	if len(failures) != 0 {
		t.Errorf("failures = %v, want none", failures)
	}
}

func TestRunDoctestsReportsMismatch(t *testing.T) {
	// The int doctest expects the wrong value; the string doctest stays correct.
	bad := strings.Replace(doctestProgram, "/// 5\n", "/// 6\n", 1)
	failures, ran, err := RunDoctests(bad)
	if err != nil {
		t.Fatalf("RunDoctests: unexpected error: %v", err)
	}
	if ran != 2 {
		t.Errorf("ran = %d, want 2", ran)
	}
	if len(failures) != 1 {
		t.Fatalf("failures = %d, want 1: %v", len(failures), failures)
	}
	f := failures[0]
	if f.Func != "add" {
		t.Errorf("Func = %q, want add", f.Func)
	}
	if f.Input != "add(2, 3)" {
		t.Errorf("Input = %q, want add(2, 3)", f.Input)
	}
	if f.Expected != "6" {
		t.Errorf("Expected = %q, want 6", f.Expected)
	}
	if f.Got != "5" {
		t.Errorf("Got = %q, want 5", f.Got)
	}
}

func TestRunDoctestsEvalErrorIsLoud(t *testing.T) {
	// The doctest calls a symbol the program does not declare: an eval error, not a
	// silent pass.
	prog := `package p

/// >>> missing(1)
/// 1
func f() int {
	return 0
}
`
	_, _, err := RunDoctests(prog)
	if err == nil {
		t.Fatalf("RunDoctests: expected an error for an undefined doctest symbol")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("error %q does not name the missing symbol", err)
	}
}
