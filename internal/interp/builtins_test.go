package interp

// These tests prove US-010 "Eval builtins and methods": the interpreter
// implements the len, append, make, and panic builtins and dispatches both
// value-receiver and pointer-receiver methods declared on goal types. The tests
// drive real parsed+resolved goal programs through the standard direct-evalExpr
// seam (ip.evalExpr(call("fn"), ip.root)) established by US-005..US-009.

import (
	"errors"
	"testing"
)

const builtinsProgram = `package main

func sliceLen() int {
	s := []int{1, 2, 3}
	return len(s)
}

func sliceAppend() int {
	s := []int{1, 2, 3}
	s = append(s, 4, 5)
	return s[4]
}

func sliceAppendLen() int {
	s := []int{1, 2, 3}
	s = append(s, 4, 5)
	return len(s)
}

func makeMapLen() int {
	m := make(map[string]int)
	m["a"] = 1
	m["b"] = 2
	return len(m)
}

func makeMapRead() int {
	m := make(map[string]int)
	m["a"] = 7
	return m["a"]
}

func makeSlice() int {
	s := make([]int, 3)
	return len(s) + s[0] + s[2]
}

func strLen() int {
	return len("hello")
}

func boom() {
	panic("kaboom")
}

func main() {}
`

func TestBuiltinLen(t *testing.T) {
	ip := newInterp(t, builtinsProgram)
	cases := []struct {
		fn   string
		want int64
	}{
		{"sliceLen", 3},
		{"sliceAppendLen", 5},
		{"makeMapLen", 2},
		{"makeSlice", 3},
		{"strLen", 5},
	}
	for _, c := range cases {
		got, err := ip.evalExpr(call(c.fn), ip.root)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.fn, err)
		}
		if got.Kind != KindInt || got.Int != c.want {
			t.Fatalf("%s = %v, want %d", c.fn, got, c.want)
		}
	}
}

func TestBuiltinAppendValue(t *testing.T) {
	ip := newInterp(t, builtinsProgram)
	got, err := ip.evalExpr(call("sliceAppend"), ip.root)
	if err != nil {
		t.Fatalf("sliceAppend: unexpected error: %v", err)
	}
	if got.Kind != KindInt || got.Int != 5 {
		t.Fatalf("sliceAppend (s[4]) = %v, want 5", got)
	}
}

func TestBuiltinMakeMapReadBack(t *testing.T) {
	ip := newInterp(t, builtinsProgram)
	got, err := ip.evalExpr(call("makeMapRead"), ip.root)
	if err != nil {
		t.Fatalf("makeMapRead: unexpected error: %v", err)
	}
	if got.Kind != KindInt || got.Int != 7 {
		t.Fatalf("makeMapRead = %v, want 7", got)
	}
}

func TestBuiltinPanicRecovered(t *testing.T) {
	ip := newInterp(t, builtinsProgram)
	_, err := ip.evalExpr(call("boom"), ip.root)
	if err == nil {
		t.Fatalf("boom: expected a panic error, got nil")
	}
	var p panicSignal
	if !errors.As(err, &p) {
		t.Fatalf("boom: error %v is not a panicSignal", err)
	}
	if p.value.Kind != KindString || p.value.Str != "kaboom" {
		t.Fatalf("panic value = %v, want \"kaboom\"", p.value)
	}
}

func TestBuiltinLenUndefinedOperand(t *testing.T) {
	// len of an int has no defined meaning — a descriptive refusal, not a panic.
	prog := newInterp(t, `package main
func badLen() int {
	x := 3
	return len(x)
}
func main() {}
`)
	if _, err := prog.evalExpr(call("badLen"), prog.root); err == nil {
		t.Fatalf("len(int): expected a descriptive error, got nil")
	}
}

const methodsProgram = `package main

type Counter struct {
	n int
}

func (c *Counter) Inc() {
	c.n += 1
}

func (c *Counter) Add(d int) {
	c.n += d
}

func (c Counter) Peek() int {
	return c.n
}

func (c Counter) Grow() {
	c.n += 100
}

func pointerReceiverMutates() int {
	c := Counter{n: 0}
	c.Inc()
	c.Inc()
	c.Add(10)
	return c.n
}

func valueReceiverReads() int {
	c := Counter{n: 42}
	return c.Peek()
}

func valueReceiverDoesNotLeak() int {
	c := Counter{n: 5}
	c.Grow()
	return c.n
}

func main() {}
`

func TestPointerReceiverMethodMutates(t *testing.T) {
	ip := newInterp(t, methodsProgram)
	got, err := ip.evalExpr(call("pointerReceiverMutates"), ip.root)
	if err != nil {
		t.Fatalf("pointerReceiverMutates: unexpected error: %v", err)
	}
	// 0, +1, +1, +10 = 12
	if got.Kind != KindInt || got.Int != 12 {
		t.Fatalf("pointerReceiverMutates = %v, want 12", got)
	}
}

func TestValueReceiverMethodReads(t *testing.T) {
	ip := newInterp(t, methodsProgram)
	got, err := ip.evalExpr(call("valueReceiverReads"), ip.root)
	if err != nil {
		t.Fatalf("valueReceiverReads: unexpected error: %v", err)
	}
	if got.Kind != KindInt || got.Int != 42 {
		t.Fatalf("valueReceiverReads = %v, want 42", got)
	}
}

func TestValueReceiverMethodDoesNotLeak(t *testing.T) {
	ip := newInterp(t, methodsProgram)
	got, err := ip.evalExpr(call("valueReceiverDoesNotLeak"), ip.root)
	if err != nil {
		t.Fatalf("valueReceiverDoesNotLeak: unexpected error: %v", err)
	}
	// Grow mutates a value-receiver COPY, so the caller's field stays 5.
	if got.Kind != KindInt || got.Int != 5 {
		t.Fatalf("valueReceiverDoesNotLeak = %v, want 5 (value receiver must not leak)", got)
	}
}
