package interp

// These tests prove US-009 "Eval composite types": the interpreter evaluates
// struct composite literals + field access, slice literals + indexing, map
// literals + indexing + key assignment, and range-for over slices and maps.
// They drive real parsed+resolved goal programs through the direct-evalExpr
// testing seam (newInterp + call, from call_test.go), asserting the returned
// values. Maps are string-keyed (the v1 value model).

import (
	"strings"
	"testing"
)

// evalFn runs the named zero-argument function and returns its single result.
func evalFn(t *testing.T, ip *Interp, name string) Value {
	t.Helper()
	v, err := ip.evalExpr(call(name), ip.root)
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", name, err)
	}
	return v
}

// evalFnErr runs the named zero-argument function expecting a descriptive error.
func evalFnErr(t *testing.T, ip *Interp, name string) error {
	t.Helper()
	_, err := ip.evalExpr(call(name), ip.root)
	if err == nil {
		t.Fatalf("%s: expected an error, got none", name)
	}
	return err
}

const compositeProgram = `package main

type Point struct {
	X int
	Y int
}

func structFields() int {
	p := Point{X: 3, Y: 4}
	return p.X + p.Y
}

func structFieldAssign() int {
	p := Point{X: 1, Y: 2}
	p.X = 10
	p.Y += 5
	return p.X + p.Y
}

func sliceIndex() int {
	s := []int{10, 20, 30}
	return s[0] + s[1] + s[2]
}

func sliceElementAssign() int {
	s := []int{1, 2, 3}
	s[1] = 20
	s[2] += 7
	return s[0] + s[1] + s[2]
}

func mapIndexAndAssign() int {
	m := map[string]int{"a": 1, "b": 2}
	m["a"] = 100
	m["c"] = 3
	return m["a"] + m["b"] + m["c"]
}

func rangeSliceValues() int {
	s := []int{1, 2, 3, 4}
	total := 0
	for _, v := range s {
		total += v
	}
	return total
}

func rangeSliceIndices() int {
	s := []int{5, 6, 7}
	sum := 0
	for i, v := range s {
		sum += i * v
	}
	return sum
}

func rangeMapValues() int {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	total := 0
	for _, v := range m {
		total += v
	}
	return total
}

func rangeMapKeys() string {
	m := map[string]int{"c": 3, "a": 1, "b": 2}
	keys := ""
	for k := range m {
		keys += k
	}
	return keys
}

func rangeBreakContinue() int {
	s := []int{1, 2, 3, 4, 5, 6}
	sum := 0
	for _, v := range s {
		if v == 2 {
			continue
		}
		if v == 5 {
			break
		}
		sum += v
	}
	return sum
}

func main() {}
`

func TestCompositeProgram(t *testing.T) {
	ip := newInterp(t, compositeProgram)

	intCases := []struct {
		fn   string
		want int64
	}{
		{"structFields", 7},
		{"structFieldAssign", 17},
		{"sliceIndex", 60},
		{"sliceElementAssign", 31},
		{"mapIndexAndAssign", 105},
		{"rangeSliceValues", 10},
		{"rangeSliceIndices", 20},
		{"rangeMapValues", 6},
		{"rangeBreakContinue", 8}, // 1 + 3 + 4 (skip 2, stop at 5)
	}
	for _, c := range intCases {
		got := evalFn(t, ip, c.fn)
		if got.Kind != KindInt || got.Int != c.want {
			t.Errorf("%s = %v, want int %d", c.fn, got, c.want)
		}
	}

	// Ranging a map visits keys in sorted (deterministic) order.
	if got := evalFn(t, ip, "rangeMapKeys"); got.Kind != KindString || got.Str != "abc" {
		t.Errorf("rangeMapKeys = %v, want %q", got, "abc")
	}
}

// TestAcceptanceCombined is the AC unit test: one program builds and reads
// structs, slices, and maps, ranges over a slice and a map, and asserts the
// collected results.
func TestAcceptanceCombined(t *testing.T) {
	const src = `package main

type Item struct {
	Name  string
	Price int
}

func total() int {
	items := []Item{
		Item{Name: "apple", Price: 3},
		Item{Name: "pear", Price: 5},
	}
	counts := map[string]int{"apple": 2, "pear": 4}

	sum := 0
	for _, it := range items {
		sum += it.Price * counts[it.Name]
	}
	return sum
}

func main() {}
`
	ip := newInterp(t, src)
	// apple: 3*2=6, pear: 5*4=20, total 26.
	if got := evalFn(t, ip, "total"); got.Kind != KindInt || got.Int != 26 {
		t.Fatalf("total = %v, want int 26", got)
	}
}

func TestCompositeErrors(t *testing.T) {
	const src = `package main

func outOfRange() int {
	s := []int{1, 2}
	return s[5]
}

func nonStringMapKey() string {
	m := map[int]string{1: "x"}
	return m[1]
}

func fieldOnNonStruct() int {
	x := 5
	return x.foo
}

func indexNonCollection() int {
	x := 5
	return x[0]
}

func main() {}
`
	ip := newInterp(t, src)

	cases := []struct {
		fn       string
		contains string
	}{
		{"outOfRange", "out of range"},
		{"nonStringMapKey", "map key must be string"},
		{"fieldOnNonStruct", "cannot select field"},
		{"indexNonCollection", "cannot index"},
	}
	for _, c := range cases {
		err := evalFnErr(t, ip, c.fn)
		if !strings.Contains(err.Error(), c.contains) {
			t.Errorf("%s error = %q, want it to contain %q", c.fn, err.Error(), c.contains)
		}
	}
}
