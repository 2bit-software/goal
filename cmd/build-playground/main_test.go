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
