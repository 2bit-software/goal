package check

import (
	"strings"
	"testing"
)

// hasCode reports whether any diagnostic carries the given code.
func hasCode(diags []Diagnostic, code string) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

// TestControlHeaderNotLiteral guards the §8 field-completeness check against a lexical
// false positive: a control-flow header whose tag/condition is a bare identifier wears
// the same `IDENT {` shape as a composite literal (`switch i {` vs `i{…}`), so it must
// not be deferred as an unresolved-literal-type. (Regression: `switch i {` reported
// `cannot verify field completeness of i{…}`.)
func TestControlHeaderNotLiteral(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"switch-bare-ident", `package x

func f(i int) string {
	switch i {
	case 0:
		return "zero"
	default:
		return "other"
	}
}
`},
		{"if-bare-ident", `package x

func f(ok bool) int {
	if ok {
		return 1
	}
	return 0
}
`},
		{"for-bare-ident", `package x

func f(more bool) {
	for more {
		more = false
	}
}
`},
		{"switch-with-label-arm", `package x

func f(s string) string {
	switch s {
	case "a":
		return "A"
	default:
		return "z"
	}
}
`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			diags, err := Analyze(c.src)
			if err != nil {
				t.Fatalf("Analyze: %v", err)
			}
			if hasCode(diags, "unresolved-literal-type") {
				for _, d := range diags {
					t.Logf("diag: [%s] %s", d.Code, d.Message)
				}
				t.Errorf("control-flow header was misread as a composite literal")
			}
		})
	}
}

// TestSliceLiteralElementTypeNotStruct guards the §8 field-completeness check against a
// lexical false positive: the element/value type of a slice/array/map literal wears the
// same `IDENT {` shape as a struct literal (`[]Entry{}` vs `Entry{}`), so an empty slice
// of a struct type must not be read as a zero-value struct construction. (Regression:
// `[]Entry{}` reported `struct literal Entry{…} omits required fields`.)
func TestSliceLiteralElementTypeNotStruct(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"empty-slice", `package x

type Entry struct {
	Name string
	Val  int
}

func f() []Entry {
	return []Entry{}
}
`},
		{"empty-array", `package x

type Entry struct {
	Name string
	Val  int
}

func f() [0]Entry {
	return [0]Entry{}
}
`},
		{"empty-map-value", `package x

type Entry struct {
	Name string
	Val  int
}

func f() map[string]Entry {
	return map[string]Entry{}
}
`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			diags, err := Analyze(c.src)
			if err != nil {
				t.Fatalf("Analyze: %v", err)
			}
			if hasCode(diags, "missing-field") {
				for _, d := range diags {
					t.Logf("diag: [%s] %s", d.Code, d.Message)
				}
				t.Errorf("a slice/array/map literal's element type was misread as a struct literal")
			}
		})
	}
}

// TestStructLiteralStillChecked confirms the control-header fix did not blind the check
// to genuine struct literals nested inside a control-flow body: a real incomplete literal
// inside a `switch`/`if` body must still be reported.
func TestStructLiteralStillChecked(t *testing.T) {
	const src = `package x

type Point struct {
	X int
	Y int
}

func f(i int) Point {
	switch i {
	default:
		return Point{X: 1}
	}
}
`
	diags, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if !hasCode(diags, "missing-field") {
		t.Errorf("incomplete struct literal inside a switch body was not reported")
	}
	for _, d := range diags {
		if d.Code == "missing-field" && !strings.Contains(d.Message, "Y") {
			t.Errorf("missing-field diagnostic did not name the omitted field Y: %s", d.Message)
		}
	}
}
