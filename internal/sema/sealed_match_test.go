package sema

import "testing"

// SEAM-CAP-3b: exhaustiveness of a type-pattern `match` over a same-package sealed
// interface is checked against the implementor registry (built from `implements`
// clauses), mirroring enum exhaustiveness.

const sealedHeader = `package p

sealed interface Node {}
type Lit struct implements Node { Val int }
type Neg struct implements Node { Inner Node }
`

func TestSealedMatchExhaustiveNoDiagnostics(t *testing.T) {
	src := sealedHeader + `
func eval(n Node) int {
	return match n {
		*Lit(l) => l.Val
		*Neg(g) => -eval(g.Inner)
	}
}
`
	if d := analyzeExhaustive(t, src); len(d) != 0 {
		t.Fatalf("exhaustive sealed match produced diagnostics: %+v", d)
	}
}

func TestSealedMatchNonExhaustiveIsError(t *testing.T) {
	src := sealedHeader + `
func eval(n Node) int {
	return match n {
		*Lit(l) => l.Val
	}
}
`
	d := analyzeExhaustive(t, src)
	if len(d) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(d), d)
	}
	if SeverityLabel(d[0].Severity) != "error" {
		t.Errorf("want Error severity, got %v", d[0].Severity)
	}
	if d[0].Code != "non-exhaustive-match" {
		t.Errorf("want code non-exhaustive-match, got %q", d[0].Code)
	}
	if got := d[0].Message; !contains(got, "*Neg") {
		t.Errorf("message should name the missing implementor `*Neg`: %q", got)
	}
}

func TestSealedMatchRestArmAccepted(t *testing.T) {
	src := sealedHeader + `
func eval(n Node) int {
	return match n {
		*Lit(l) => l.Val
		_ => 0
	}
}
`
	if d := analyzeExhaustive(t, src); len(d) != 0 {
		t.Fatalf("sealed match with `_` rest-arm produced diagnostics: %+v", d)
	}
}

// TestSealedMatchUnresolvedDefers proves a type-pattern match whose concrete type is
// not registered by any same-package sealed interface defers (a Warning, not a false
// Error) — the boundary CAP-3c (cross-package) will close.
func TestSealedMatchUnresolvedDefers(t *testing.T) {
	const src = `package p

func eval(n any) int {
	return match n {
		*Foo(f) => 1
	}
}
`
	d := analyzeExhaustive(t, src)
	if len(d) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(d), d)
	}
	if SeverityLabel(d[0].Severity) != "warning" {
		t.Errorf("want Warning severity, got %v", d[0].Severity)
	}
	if d[0].Code != "unresolved-match-sealed" {
		t.Errorf("want code unresolved-match-sealed, got %q", d[0].Code)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
