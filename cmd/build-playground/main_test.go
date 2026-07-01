package main

import (
	"strings"
	"testing"
)

// A non-exhaustive match is a checker-only diagnostic: the backend transpiles it
// cleanly, so it exercises the sema.Analyze verification path (not backend).
const matchSource = `package traffic

enum Light {
	Red
	Yellow
	Green
}

func name(l Light) string {
	return match l {
		Light.Red => "red"
		Light.Yellow => "yellow"
	}
}
`

// The located diagnostic the live checker renders for matchSource with the
// feature's SourceName "traffic.goal".
const matchDiagnostic = "traffic.goal:10:9: error: [non-exhaustive-match] non-exhaustive `match` on enum `Light`: missing variant `Light.Green` — handle it, or add a `_` rest-arm to dismiss the rest"

// The unsafe-default example is a backend rejection (its locked block is the
// "backend: …" form), so it must verify via the backend.Transpile fallback.
const defaultsSource = `package cache

type Store struct {
	name    string
	entries map[string]int
}

func newStore(name string) Store {
	return Store{name: name, ...defaults}
}
`

const defaultsRejection = "backend: `...defaults` at 9:27 cannot default field `entries` of type `map[string]int`: a nil map panics on write — set it explicitly (e.g. `map[string]int{}`)"

// A checker error feature whose locked block matches the live checker render
// verifies cleanly, rendered against the feature's own SourceName.
func TestVerifyErrorChecker(t *testing.T) {
	if err := verify(matchSource, matchDiagnostic, "error", "traffic.goal"); err != nil {
		t.Fatalf("checker error feature should verify, got: %v", err)
	}
}

// A checker error feature whose locked text is wrong is rejected on the checker
// path (rendered diagnostic does not equal the locked block).
func TestVerifyErrorCheckerWrongText(t *testing.T) {
	wrong := "traffic.goal:10:9: error: [non-exhaustive-match] this is not what the checker says"
	err := verify(matchSource, wrong, "error", "traffic.goal")
	if err == nil {
		t.Fatal("wrong locked text should fail verification, got nil")
	}
	if !strings.Contains(err.Error(), "does not match live checker") {
		t.Fatalf("expected a checker-mismatch error, got: %v", err)
	}
}

// The unsafe-default backend example (backend-format locked block) still verifies
// via the backend.Transpile fallback, unchanged by the checker path.
func TestVerifyErrorBackendFallback(t *testing.T) {
	if err := verify(defaultsSource, defaultsRejection, "error", "defaults_unsafe.goal"); err != nil {
		t.Fatalf("backend error feature should verify via fallback, got: %v", err)
	}
}

// A doctest whose expected value is wrong: the generated _test.go fails, so this
// is a genuine doctest-failure feature. Its locked block is the normalized failure
// message the toolchain prints.
const failingDoctestSource = `package mathx

/// Squares an int.
/// >>> square(3)
/// 8
func square(x int) int {
	return x * x
}
`

const failingDoctestBlock = "doctest square: got 9, want 8"

// A doctest whose expected value is correct: the generated _test.go passes, so it
// must NOT be accepted as a doctest-failure feature.
const passingDoctestSource = `package mathx

/// Squares an int.
/// >>> square(3)
/// 9
func square(x int) int {
	return x * x
}
`

// A doctest-failure feature whose locked block matches the live test failure
// (compiled and run) verifies cleanly.
func TestVerifyDoctestFailureAccepted(t *testing.T) {
	if err := verify(failingDoctestSource, failingDoctestBlock, "doctest-failure", "mathx.goal"); err != nil {
		t.Fatalf("failing doctest feature should verify, got: %v", err)
	}
}

// A passing doctest must be rejected as a doctest-failure feature: its generated
// test passes, so the shown failure would be a lie.
func TestVerifyDoctestFailurePassingRejected(t *testing.T) {
	err := verify(passingDoctestSource, failingDoctestBlock, "doctest-failure", "mathx.goal")
	if err == nil {
		t.Fatal("a passing doctest must not be accepted as a doctest-failure feature")
	}
	if !strings.Contains(err.Error(), "expected the generated doctest to fail") {
		t.Fatalf("expected a passing-doctest rejection, got: %v", err)
	}
}

// A doctest-failure feature whose locked block does not match the observed failure
// is rejected, so the shown failure cannot drift from the toolchain.
func TestVerifyDoctestFailureWrongText(t *testing.T) {
	err := verify(failingDoctestSource, "doctest square: got 9, want 7", "doctest-failure", "mathx.goal")
	if err == nil {
		t.Fatal("wrong locked failure text should fail verification, got nil")
	}
	if !strings.Contains(err.Error(), "does not match the live test failure") {
		t.Fatalf("expected a failure-mismatch error, got: %v", err)
	}
}

// normalizeDoctestFailure strips the harness prefix/framing down to the authored
// `doctest …: got …, want …` message(s).
func TestNormalizeDoctestFailure(t *testing.T) {
	raw := "--- FAIL: TestDoctest_square_1 (0.00s)\n    feature_test.go:9: doctest square: got 9, want 8\nFAIL\nFAIL\tgoal\t0.2s\n"
	if got := normalizeDoctestFailure(raw); got != "doctest square: got 9, want 8" {
		t.Fatalf("normalized failure = %q, want %q", got, "doctest square: got 9, want 8")
	}
	if got := normalizeDoctestFailure("ok\tgoal\t0.1s\n"); got != "" {
		t.Fatalf("passing output should normalize to empty, got %q", got)
	}
}

// isCheckerDiagnostic routes located checker blocks to the checker and
// backend-format blocks to the backend fallback.
func TestIsCheckerDiagnostic(t *testing.T) {
	if !isCheckerDiagnostic(matchDiagnostic, "traffic.goal") {
		t.Error("located diagnostic should be recognized as a checker diagnostic")
	}
	if isCheckerDiagnostic(defaultsRejection, "defaults_unsafe.goal") {
		t.Error("backend rejection should not be recognized as a checker diagnostic")
	}
	if isCheckerDiagnostic(matchDiagnostic, "") {
		t.Error("empty SourceName should never match")
	}
}
