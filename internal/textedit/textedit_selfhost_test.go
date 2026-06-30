package textedit

import "testing"

// These tests are self-contained (stdlib testing only, no fixtures) so they can
// serve as the US-007 parity oracle: they run here against the legacy textedit
// package under `task check`, and the same file is replayed against the
// goal-sourced internal/compiler/textedit by internal/selfhost's
// TestPortedTexteditPackage. Identical pinned outputs on both sides prove the
// goal-built textedit produces the same resulting text as the legacy package for
// the same edits.

func TestSpliceApplies(t *testing.T) {
	cases := []struct {
		name   string
		src    string
		lo, hi int
		reps   []Replacement
		want   string
	}{
		{"single replacement", "hello world", 0, 11, []Replacement{{6, 11, "there"}}, "hello there"},
		{"no replacements is a slice", "abcdef", 1, 4, nil, "bcd"},
		{"unsorted reps applied left to right", "0123456789", 0, 10,
			[]Replacement{{4, 6, "X"}, {1, 2, "Y"}}, "0Y23X6789"},
		{"overlapping later rep is skipped", "0123456789", 0, 10,
			[]Replacement{{0, 5, "A"}, {3, 6, "B"}}, "A56789"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Splice(c.src, c.lo, c.hi, c.reps); got != c.want {
				t.Fatalf("Splice(%q,%d,%d,%v) = %q, want %q", c.src, c.lo, c.hi, c.reps, got, c.want)
			}
		})
	}
}

func TestIsLineStart(t *testing.T) {
	cases := []struct {
		src  string
		p    int
		want bool
	}{
		{"  abc", 2, true},
		{"x abc", 2, false},
		{"a\n  b", 4, true},
		{"abc", 0, true},
	}
	for _, c := range cases {
		if got := IsLineStart(c.src, c.p); got != c.want {
			t.Errorf("IsLineStart(%q,%d) = %v, want %v", c.src, c.p, got, c.want)
		}
	}
}

func TestNextNewline(t *testing.T) {
	cases := []struct {
		src  string
		p    int
		want int
	}{
		{"ab\ncd", 0, 2},
		{"abc", 0, 3},
		{"a\nb\nc", 2, 3},
	}
	for _, c := range cases {
		if got := NextNewline(c.src, c.p); got != c.want {
			t.Errorf("NextNewline(%q,%d) = %d, want %d", c.src, c.p, got, c.want)
		}
	}
}

func TestLeadIdent(t *testing.T) {
	cases := []struct {
		s, want string
	}{
		{"parse(x)", "parse"},
		{"_foo.bar", "_foo"},
		{"+x", ""},
		{"abc", "abc"},
	}
	for _, c := range cases {
		if got := LeadIdent(c.s); got != c.want {
			t.Errorf("LeadIdent(%q) = %q, want %q", c.s, got, c.want)
		}
	}
}

func TestIsIdent(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"foo", true},
		{"_x", true},
		{"1abc", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsIdent(c.s); got != c.want {
			t.Errorf("IsIdent(%q) = %v, want %v", c.s, got, c.want)
		}
	}
}

func TestSplitAssign(t *testing.T) {
	cases := []struct {
		s        string
		wantName string
		wantRHS  string
		wantOK   bool
	}{
		{"x := 5", "x", "5", true},
		{"  a, b := f()  ", "a, b", "f()", true},
		{"no assign here", "", "no assign here", false},
	}
	for _, c := range cases {
		name, rhs, ok := SplitAssign(c.s)
		if name != c.wantName || rhs != c.wantRHS || ok != c.wantOK {
			t.Errorf("SplitAssign(%q) = (%q,%q,%v), want (%q,%q,%v)",
				c.s, name, rhs, ok, c.wantName, c.wantRHS, c.wantOK)
		}
	}
}

func TestIsStmtKeyword(t *testing.T) {
	for _, kw := range []string{"return", "for", "match", "assert", "enum", "package"} {
		if !IsStmtKeyword(kw) {
			t.Errorf("IsStmtKeyword(%q) = false, want true", kw)
		}
	}
	for _, id := range []string{"foo", "matcher", "", "Return"} {
		if IsStmtKeyword(id) {
			t.Errorf("IsStmtKeyword(%q) = true, want false", id)
		}
	}
}

func TestBaseType(t *testing.T) {
	cases := []struct {
		t, want string
	}{
		{"*pkg.Foo", "Foo"},
		{"  *Bar ", "Bar"},
		{"Baz", "Baz"},
		{"a.b.C", "C"},
	}
	for _, c := range cases {
		if got := BaseType(c.t); got != c.want {
			t.Errorf("BaseType(%q) = %q, want %q", c.t, got, c.want)
		}
	}
}

func TestZeroLit(t *testing.T) {
	cases := []struct {
		typ   string
		decls map[string]string
		want  string
	}{
		{"int", nil, "0"},
		{"string", nil, `""`},
		{"bool", nil, "false"},
		{"*Foo", nil, "nil"},
		{"[]int", nil, "nil"},
		{"map[string]int", nil, "nil"},
		{"chan int", nil, "nil"},
		{"error", nil, "nil"},
		{"any", nil, "nil"},
		{"[3]int", nil, "[3]int{}"},
		{"Foo", nil, "Foo{}"},
		{"Role", map[string]string{"Role": "int"}, "0"},
		{"S", map[string]string{"S": "struct"}, "S{}"},
		{"I", map[string]string{"I": "interface"}, "nil"},
	}
	for _, c := range cases {
		if got := ZeroLit(c.typ, c.decls, 0); got != c.want {
			t.Errorf("ZeroLit(%q,%v) = %q, want %q", c.typ, c.decls, got, c.want)
		}
	}
}
