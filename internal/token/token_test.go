package token

import "testing"

// TestKindStringRoundTrip asserts that every operator and keyword kind's String()
// spelling round-trips back to the identical Kind via Lookup.
func TestKindStringRoundTrip(t *testing.T) {
	for k := operatorBeg + 1; k < operatorEnd; k++ {
		name := k.String()
		got, ok := Lookup(name)
		if !ok {
			t.Errorf("operator %v (%q): Lookup reported not found", k, name)
			continue
		}
		if got != k {
			t.Errorf("operator round-trip: Lookup(%q) = %v, want %v", name, got, k)
		}
	}
	for k := keywordBeg + 1; k < keywordEnd; k++ {
		name := k.String()
		got, ok := Lookup(name)
		if !ok {
			t.Errorf("keyword %v (%q): Lookup reported not found", k, name)
			continue
		}
		if got != k {
			t.Errorf("keyword round-trip: Lookup(%q) = %v, want %v", name, got, k)
		}
	}
}

// TestGoalSpecificLexemes pins the spellings of the goal-specific lexemes the splice
// approach faked, ensuring each is exactly one kind.
func TestGoalSpecificLexemes(t *testing.T) {
	cases := []struct {
		spelling string
		want     Kind
	}{
		{"?", QUESTION},
		{"=>", FAT_ARROW},
		{"...", ELLIPSIS},
		{"match", MATCH},
		{"enum", ENUM},
		{"assert", ASSERT},
	}
	for _, c := range cases {
		got, ok := Lookup(c.spelling)
		if !ok || got != c.want {
			t.Errorf("Lookup(%q) = (%v, %v), want (%v, true)", c.spelling, got, ok, c.want)
		}
		if c.want.String() != c.spelling {
			t.Errorf("%v.String() = %q, want %q", c.want, c.want.String(), c.spelling)
		}
	}
	// '=>' must be one token, not '=' (ASSIGN) followed by '>' (GTR).
	if FAT_ARROW == ASSIGN || FAT_ARROW == GTR {
		t.Error("FAT_ARROW must be a distinct kind from ASSIGN and GTR")
	}
}

// TestContextualKeywordsAreNotKeywords asserts that implements/sealed/from/derive are
// not reserved — they lex as IDENT and are not found by Lookup.
func TestContextualKeywordsAreNotKeywords(t *testing.T) {
	for _, name := range []string{"implements", "sealed", "from", "derive"} {
		if IsKeyword(name) {
			t.Errorf("IsKeyword(%q) = true, want false (contextual keyword)", name)
		}
		if k, ok := Lookup(name); ok {
			t.Errorf("Lookup(%q) = (%v, true), want (ILLEGAL, false)", name, k)
		}
	}
}

// TestLookupUnknown asserts unknown spellings return (ILLEGAL, false) without panic.
func TestLookupUnknown(t *testing.T) {
	if k, ok := Lookup("definitelyNotAToken"); ok || k != ILLEGAL {
		t.Errorf("Lookup(unknown) = (%v, %v), want (ILLEGAL, false)", k, ok)
	}
	if k, ok := Lookup(""); ok || k != ILLEGAL {
		t.Errorf("Lookup(empty) = (%v, %v), want (ILLEGAL, false)", k, ok)
	}
}

// TestKindClassPredicates spot-checks the range predicates.
func TestKindClassPredicates(t *testing.T) {
	if !IDENT.IsLiteral() || IDENT.IsKeyword() || IDENT.IsOperator() {
		t.Error("IDENT should classify only as literal")
	}
	if !ELLIPSIS.IsOperator() || ELLIPSIS.IsKeyword() || ELLIPSIS.IsLiteral() {
		t.Error("ELLIPSIS should classify only as operator")
	}
	if !MATCH.IsKeyword() || MATCH.IsOperator() || MATCH.IsLiteral() {
		t.Error("MATCH should classify only as keyword")
	}
}

// TestKindStringOutOfRange asserts an undefined kind renders deterministically.
func TestKindStringOutOfRange(t *testing.T) {
	if got := Kind(9999).String(); got != "token(9999)" {
		t.Errorf("Kind(9999).String() = %q, want %q", got, "token(9999)")
	}
}

// TestPosOrdering asserts Pos orders by byte offset: earlier before later, and
// neither equal nor greater positions are Less.
func TestPosOrdering(t *testing.T) {
	a := Pos{Offset: 1, Line: 1, Col: 2}
	b := Pos{Offset: 2, Line: 1, Col: 3}
	if !a.Less(b) {
		t.Errorf("%v.Less(%v) = false, want true", a, b)
	}
	if b.Less(a) {
		t.Errorf("%v.Less(%v) = true, want false", b, a)
	}
	c := Pos{Offset: 5, Line: 3, Col: 1}
	if c.Less(c) {
		t.Errorf("%v.Less(itself) = true, want false (irreflexive)", c)
	}
	// A later line with the same offset (degenerate) still orders by offset only.
	d := Pos{Offset: 5, Line: 9, Col: 9}
	if c.Less(d) || d.Less(c) {
		t.Error("equal-offset positions must not order either way")
	}
}

// TestPosString checks the diagnostic rendering.
func TestPosString(t *testing.T) {
	if got := (Pos{Offset: 10, Line: 4, Col: 7}).String(); got != "4:7" {
		t.Errorf("Pos.String() = %q, want %q", got, "4:7")
	}
}
